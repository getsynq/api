"""
Credential resolution for the governance-as-code tool.

Auth is kept SEPARATE from the governance definition file — the YAML never holds
secrets. Credentials come from CLI flags or the environment, in priority order:

  1. A bearer token       (--token             / QUALITY_TOKEN)
  2. OAuth2 client creds  (--client-id/-secret / QUALITY_CLIENT_ID + QUALITY_CLIENT_SECRET)
  3. A cached browser login for this tool (`governance-as-code login`)

The cached login is stored under this tool's own config directory
(``$XDG_CONFIG_HOME/coalesce-quality`` by default, override with
``GOVERNANCE_HOME``) — not shared with other tools.

The legacy ``SYNQ_*`` names are also accepted for back-compat; ``QUALITY_*`` wins.
The ``synq.io`` endpoints are the live product addresses.
"""

import base64
import hashlib
import http.server
import json
import os
import secrets
import threading
import time
import urllib.parse
import webbrowser
from datetime import datetime, timezone

import grpc
import requests


def config_home():
    base = os.environ.get("GOVERNANCE_HOME") or os.path.join(
        os.environ.get("XDG_CONFIG_HOME", os.path.expanduser("~/.config")), "coalesce-quality"
    )
    os.makedirs(base, exist_ok=True)
    return base


def _token_url(endpoint):
    return f"https://{endpoint}/oauth2/token"


def _paths(endpoint):
    host = endpoint.split(":")[0]
    d = config_home()
    return os.path.join(d, f"token_{host}.json"), os.path.join(d, f"client_{host}.json")


class _Bearer(grpc.AuthMetadataPlugin):
    """Adapts a zero-arg token() callable into gRPC call credentials."""

    def __init__(self, token_fn, description):
        self._token_fn = token_fn
        self.description = description

    def __call__(self, context, callback):
        try:
            callback([("authorization", f"Bearer {self._token_fn()}")], None)
        except Exception as e:
            callback(None, e)


# ---- method 1: static token ------------------------------------------------

def _static(token):
    return _Bearer(lambda: token, "bearer token")


# ---- method 2: client credentials ------------------------------------------

def _client_credentials(endpoint, client_id, client_secret):
    state = {"tok": None, "exp": 0.0}

    def token():
        if state["tok"] and time.time() < state["exp"] - 60:
            return state["tok"]
        r = requests.post(_token_url(endpoint), data={
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
        })
        r.raise_for_status()
        b = r.json()
        state["tok"] = b["access_token"]
        state["exp"] = time.time() + b.get("expires_in", 3600)
        return state["tok"]

    return _Bearer(token, "client credentials")


# ---- method 3: cached browser login ----------------------------------------

def _discover(endpoint):
    r = requests.get(f"https://{endpoint.split(':')[0]}/.well-known/oauth-authorization-server")
    r.raise_for_status()
    return r.json()


def _cached_login(endpoint):
    token_path, client_path = _paths(endpoint)
    if not os.path.exists(token_path):
        return None

    def load():
        with open(token_path) as f:
            return json.load(f)

    def expired(tok):
        exp = tok.get("expires_at")
        if not exp:
            return True
        try:
            return datetime.now(timezone.utc) >= datetime.fromisoformat(exp.replace("Z", "+00:00"))
        except ValueError:
            return True

    def token():
        tok = load()
        if not expired(tok):
            return tok["access_token"]
        refresh = tok.get("refresh_token")
        if not refresh or not os.path.exists(client_path):
            raise RuntimeError("cached login expired; run `governance-as-code login` again")
        with open(client_path) as f:
            client_id = json.load(f)["client_id"]
        r = requests.post(_token_url(endpoint), data={
            "grant_type": "refresh_token", "client_id": client_id, "refresh_token": refresh,
        })
        r.raise_for_status()
        b = r.json()
        tok["access_token"] = b["access_token"]
        if b.get("refresh_token"):
            tok["refresh_token"] = b["refresh_token"]
        if b.get("expires_in"):
            tok["expires_at"] = datetime.fromtimestamp(time.time() + b["expires_in"], timezone.utc).isoformat()
        with open(token_path, "w") as f:
            json.dump(tok, f, indent=2)
        return tok["access_token"]

    return _Bearer(token, "cached browser login")


def login(endpoint):
    """Interactive browser login (OAuth2 authorization-code + PKCE). Caches the
    token + dynamically-registered client under this tool's config directory."""
    meta = _discover(endpoint)
    token_path, client_path = _paths(endpoint)

    # Bind a local callback server on the first free port.
    ports = [19870, 19871, 19872]
    httpd = None
    for p in ports:
        try:
            httpd = http.server.HTTPServer(("127.0.0.1", p), _CallbackHandler)
            redirect_uri = f"http://127.0.0.1:{p}/callback"
            break
        except OSError:
            continue
    if httpd is None:
        raise SystemExit(f"could not bind a callback port ({ports})")

    # Reuse or dynamically register a public client.
    client_id = None
    if os.path.exists(client_path):
        with open(client_path) as f:
            client_id = json.load(f).get("client_id")
    if not client_id:
        # Dynamic client registration (RFC 7591). The metadata below is what the
        # consent screen shows — keep client_name/client_uri friendly. logo_uri
        # can be added if you host a logo the IdP can fetch.
        reg = requests.post(meta["registration_endpoint"], json={
            "client_name": "Coalesce Quality — governance-as-code (CLI)",
            "client_uri": "https://coalesce.io",
            "logo_uri": "https://schemas.synq.io/coalesce-quality-logo.svg",
            "redirect_uris": [f"http://127.0.0.1:{p}/callback" for p in ports],
            "grant_types": ["authorization_code", "refresh_token"],
            "response_types": ["code"],
            "token_endpoint_auth_method": "none",
            "application_type": "native",
        })
        reg.raise_for_status()
        client_id = reg.json()["client_id"]
        with open(client_path, "w") as f:
            json.dump({"client_id": client_id}, f, indent=2)

    verifier = base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b"=").decode()
    challenge = base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
    state = secrets.token_urlsafe(16)
    auth_url = meta["authorization_endpoint"] + "?" + urllib.parse.urlencode({
        "response_type": "code", "client_id": client_id, "redirect_uri": redirect_uri,
        "state": state, "code_challenge": challenge, "code_challenge_method": "S256",
    })

    print(f"Opening browser to sign in to Coalesce Quality ({endpoint})...")
    print(f"If it doesn't open, visit:\n{auth_url}\n")
    webbrowser.open(auth_url)
    httpd.handle_request()  # serves exactly one /callback
    result = getattr(httpd, "result", {})
    if result.get("state") != state or not result.get("code"):
        raise SystemExit("login failed (state mismatch or no code)")

    tok = requests.post(_token_url(endpoint), data={
        "grant_type": "authorization_code", "code": result["code"],
        "redirect_uri": redirect_uri, "client_id": client_id, "code_verifier": verifier,
    })
    tok.raise_for_status()
    b = tok.json()
    expires_at = datetime.fromtimestamp(time.time() + b.get("expires_in", 3600), timezone.utc).isoformat()
    with open(token_path, "w") as f:
        json.dump({
            "access_token": b["access_token"],
            "refresh_token": b.get("refresh_token", ""),
            "token_type": b.get("token_type", "Bearer"),
            "expires_at": expires_at,
        }, f, indent=2)
    print(f"Signed in. Token cached at {token_path}")


class _CallbackHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        q = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
        self.server.result = {"code": q.get("code", [None])[0], "state": q.get("state", [None])[0]}
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(b"<html><body><h2>Signed in.</h2>You can close this tab.</body></html>")

    def log_message(self, *_):
        pass


# ---- resolution ------------------------------------------------------------

def _env(*names):
    for n in names:
        v = os.getenv(n)
        if v:
            return v
    return None


def resolve(endpoint, token=None, client_id=None, client_secret=None):
    # QUALITY_* is preferred; SYNQ_* accepted for back-compat.
    token = token or _env("QUALITY_TOKEN", "SYNQ_TOKEN")
    client_id = client_id or _env("QUALITY_CLIENT_ID", "SYNQ_CLIENT_ID")
    client_secret = client_secret or _env("QUALITY_CLIENT_SECRET", "SYNQ_CLIENT_SECRET")
    if token:
        return _static(token)
    if client_id and client_secret:
        return _client_credentials(endpoint, client_id, client_secret)
    cached = _cached_login(endpoint)
    if cached:
        return cached
    raise SystemExit(
        "no credentials: set QUALITY_TOKEN, or QUALITY_CLIENT_ID + QUALITY_CLIENT_SECRET, "
        "or run `governance-as-code login`."
    )


def channel(endpoint, auth_plugin):
    creds = grpc.composite_channel_credentials(
        grpc.ssl_channel_credentials(), grpc.metadata_call_credentials(auth_plugin))
    ch = grpc.secure_channel(f"{endpoint}:443", creds, options=(("grpc.default_authority", endpoint),))
    grpc.channel_ready_future(ch).result(timeout=10)
    return ch
