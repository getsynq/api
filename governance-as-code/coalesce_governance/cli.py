#!/usr/bin/env python3
# PYTHON_ARGCOMPLETE_OK
"""
governance-as-code — manage Coalesce Quality data products, owners, ownership and
alert routing declaratively from a YAML file, via the public API.

Commands:
  status   (default) Live workspace summary + next steps. Running with no command
           shows this, not help text.
  login    Sign in via the browser and cache a token for this tool.
  plan     Validate the file and show what apply would change (no writes to the API).
  apply    Apply the file (create/update). Human runs get a confirm prompt;
           agents pass --yes.
  prune    Delete the resources the file manages.
  export   Read products + owners from the workspace into a YAML file.
  pull     Refresh the fields of resources listed in a file from the API,
           preserving comments and formatting.

Identity (as in the API): every resource has a UUID `id`. Omit it and the tool
either adopts an existing resource with the same title, or generates one — either
way it writes the id back to the file so future runs are idempotent.

Regions: --region eu|us (APAC coming soon), or --endpoint to override.
Auth (never in the definition file): --token / QUALITY_TOKEN, or
--client-id/--client-secret / QUALITY_CLIENT_ID+SECRET, or a cached `login`.
Agentic use: `apply --yes`, `plan --json`; errors are structured on stdout;
exit 0 = ok, 1 = error, 2 = bad flags.
"""

import argparse
import json
import os
import sys
import uuid

try:
    import argcomplete  # optional: shell tab-completion
except ImportError:
    argcomplete = None

from . import auth
from . import api

SCHEMA_URL = "https://schemas.synq.io/governance-as-code/draft/governance.schema.json"


def _load_schema():
    from importlib.resources import files
    return json.loads(files(__package__).joinpath("governance.schema.json").read_text())

REGIONS = {"eu": "developer.synq.io", "us": "api.us.synq.io"}
COMING_SOON = {"apac"}


class CliError(Exception):
    """A user-facing error; reported as structured output with exit code 1."""


def resolve_endpoint(args):
    if args.endpoint:
        return args.endpoint
    if os.getenv("API_ENDPOINT"):
        return os.getenv("API_ENDPOINT")
    region = (args.region or "eu").lower()
    if region in COMING_SOON:
        raise CliError(f"the {region.upper()} region is not yet available")
    if region not in REGIONS:
        raise CliError(f"unknown region {region!r}; choose from: {', '.join(REGIONS)}")
    return REGIONS[region]


# ---------------------------------------------------------------- load + validate

def _yaml_rt():
    from ruamel.yaml import YAML
    y = YAML()
    y.preserve_quotes = True
    return y


def load_doc(path):
    """Load the file for round-trip editing (comment-preserving) and validate it."""
    if not os.path.exists(path):
        raise CliError(f"file not found: {path}")
    with open(path) as f:
        doc = _yaml_rt().load(f) or {}
    try:
        import jsonschema
        jsonschema.validate(json.loads(json.dumps(doc)), _load_schema())
    except ImportError:
        print("warning: jsonschema not installed; skipping pre-validation", file=sys.stderr)
    except Exception as e:  # jsonschema.ValidationError
        raise CliError(f"config validation failed: {getattr(e, 'message', e)}")
    check_unique_ids(doc)
    return doc


def check_unique_ids(doc):
    """Reject a file that reuses the same id for two resources — ids are the
    identity, so a duplicate would make two resources fight over one."""
    seen = {}
    def note(kind, node):
        i = node.get("id")
        if not i:
            return
        label = f"{kind} {node.get('title') or (node.get('query') or {}).get('name') or i!r}"
        if i in seen:
            raise CliError(f"duplicate id {i}: used by both '{seen[i]}' and '{label}'")
        seen[i] = label
    for p in doc.get("dataproducts", []):
        note("dataproduct", p)
    for o in doc.get("owners", []):
        note("owner", o)
        for os_ in o.get("ownerships", []):
            note("ownership", os_)


def _title_index(resources):
    idx = {}
    for r in resources:
        idx.setdefault(r.title, []).append(r.id)
    return idx


def _adopt_or_generate(node, kind, title_index, generate):
    """Set node['id'] if missing: adopt a unique existing resource by title, else
    (generate=True) mint a UUID. Returns True if the node was changed."""
    if node.get("id"):
        return False
    matches = title_index.get(node.get("title"), [])
    if len(matches) == 1:
        _set_id(node, matches[0])
        return True
    if len(matches) > 1:
        raise CliError(f"multiple existing {kind} titled {node.get('title')!r}; set an explicit `id`")
    if not generate:
        return False
    _set_id(node, str(uuid.uuid4()))
    return True


def _set_id(node, value):
    # Put `id` first for tidy files (ruamel CommentedMap supports insert).
    try:
        node.insert(0, "id", value)
    except (AttributeError, TypeError):
        node["id"] = value


def resolve_ids(doc, path, client, generate=True):
    """Ensure every resource has an id (adopt-by-title or generate), writing the
    file back if anything changed. Returns the number of ids assigned."""
    prod_titles = _title_index(client.list_products())
    owner_titles = _title_index(client.list_owners())
    changed = 0
    for p in doc.get("dataproducts", []):
        changed += _adopt_or_generate(p, "data products", prod_titles, generate)
    for o in doc.get("owners", []):
        changed += _adopt_or_generate(o, "owners", owner_titles, generate)
        for os_ in o.get("ownerships", []):
            if not os_.get("id") and generate:
                _set_id(os_, str(uuid.uuid4()))
                changed += 1
    if changed:
        with open(path, "w") as f:
            _yaml_rt().dump(doc, f)
        print(f"Assigned {changed} id(s) and wrote {path}")
    return changed


def _product_ref_resolver(doc):
    """Resolve an ownership `dataproduct` value (a product id or title) to an id."""
    by_title, ids = {}, set()
    for p in doc.get("dataproducts", []):
        if p.get("id"):
            ids.add(p["id"])
            by_title[p.get("title")] = p["id"]

    def resolve(value):
        if value in ids:
            return value
        if value in by_title:
            return by_title[value]
        return value  # assume an existing workspace product id

    return resolve


# ---------------------------------------------------------------- diff engine

def _contacts_norm(contacts):
    return [tuple(sorted((k, tuple(v) if isinstance(v, list) else v) for k, v in c.items())) for c in contacts]


def _alert_norm(a):
    return (
        tuple(sorted(a.get("severities", []))),
        bool(a.get("notify_upstream", False)),
        bool(a.get("allow_sql_test_audit_link", False)),
        json.dumps(a.get("ongoing", "disabled"), sort_keys=True),
    )


def plan(doc, client):
    actions = []
    resolve_ref = _product_ref_resolver(doc)

    products = doc.get("dataproducts", [])
    pids = {p["id"]: p for p in products}
    current = client.get_products(pids.keys())
    for pid, p in pids.items():
        cur = current.get(pid)
        if cur is None:
            actions.append({"kind": "dataproduct", "id": pid, "title": p["title"], "action": "create", "changed": ["*"]})
            continue
        changed = []
        if p["title"] != cur.title:
            changed.append("title")
        if "folder" in p and p["folder"] != cur.folder:
            changed.append("folder")
        if "priority" in p and p["priority"] != api.priority_from_proto(cur.priority):
            changed.append("priority")
        if p["resolver_ql"].strip() != api.rendered_resolver_ql(cur).strip():
            changed.append("definition")
        actions.append({"kind": "dataproduct", "id": pid, "title": p["title"],
                        "action": "update" if changed else "unchanged", "changed": changed})

    owners = doc.get("owners", [])
    oids = {o["id"]: o for o in owners}
    cur_owners = client.get_owners(oids.keys())
    for oid, o in oids.items():
        cur = cur_owners.get(oid)
        if cur is None:
            actions.append({"kind": "owner", "id": oid, "title": o["title"], "action": "create", "changed": ["*"]})
        else:
            changed = []
            if o["title"] != cur.title:
                changed.append("title")
            if "contacts" in o and _contacts_norm(_plain(o["contacts"])) != _contacts_norm([api.contact_from_proto(c) for c in cur.contacts]):
                changed.append("contacts")
            actions.append({"kind": "owner", "id": oid, "title": o["title"],
                            "action": "update" if changed else "unchanged", "changed": changed})
        cur_os = {os_.id: os_ for os_ in (client.list_ownerships(oid) if cur is not None else [])}
        for os_spec in o.get("ownerships", []):
            osid = os_spec["id"]
            label = (os_spec.get("query") or {}).get("name") or os_spec.get("dataproduct") or osid
            c = cur_os.get(osid)
            if c is None:
                actions.append({"kind": "ownership", "id": osid, "owner_id": oid, "title": label, "action": "create", "changed": ["*"]})
                continue
            changed = []
            if "dataproduct" in os_spec:
                if not c.selection.HasField("dataproduct_id") or c.selection.dataproduct_id != resolve_ref(os_spec["dataproduct"]):
                    changed.append("selection")
            else:
                desired_q = os_spec["query"]["resolver_ql"].strip()
                cur_q = c.selection.query.rendered_resolver_ql.strip() if c.selection.HasField("query") else ""
                if desired_q != cur_q:
                    changed.append("selection")
            if _alert_norm(_plain(os_spec.get("alert", {}))) != _alert_norm(api.alert_from_proto(c.alert)):
                changed.append("alert")
            actions.append({"kind": "ownership", "id": osid, "owner_id": oid, "title": label,
                            "action": "update" if changed else "unchanged", "changed": changed})
    return actions


def _plain(x):
    """ruamel CommentedMap/Seq -> plain dict/list for comparison/serialisation."""
    return json.loads(json.dumps(x))


def print_plan(actions):
    icon = {"create": "+", "update": "~", "unchanged": "=", "delete": "-"}
    counts = {}
    for a in actions:
        counts[a["action"]] = counts.get(a["action"], 0) + 1
    for a in actions:
        if a["action"] == "unchanged":
            continue
        detail = "" if a["action"] in ("create", "delete") else f"  ({', '.join(a['changed'])})"
        print(f"  {icon[a['action']]} {a['kind']:<11} {a['title']}{detail}")
    parts = [f"{counts.get(k, 0)} {k}" for k in ("create", "update", "unchanged")]
    print(f"\nPlan: {', '.join(parts)}  (of {len(actions)} resources)")
    return counts


# ---------------------------------------------------------------- apply/prune

def do_apply(doc, client, actions, assume_yes):
    to_write = [a for a in actions if a["action"] in ("create", "update")]
    if not to_write:
        print("Nothing to apply — workspace already matches the file.")
        return
    confirm_or_die(f"Apply {len(to_write)} change(s)?", assume_yes)
    wanted = {a["id"] for a in to_write}
    resolve_ref = _product_ref_resolver(doc)

    for p in doc.get("dataproducts", []):
        if p["id"] not in wanted:
            continue
        client.upsert_product(p["id"], title=p["title"], folder=p.get("folder"),
                              priority=p.get("priority"), resolver_ql=p["resolver_ql"],
                              part_id=str(uuid.uuid5(uuid.UUID(p["id"]), "part")))
        print(f"  applied dataproduct {p['title']}  (members={client.count_members(p['id'])})")

    for o in doc.get("owners", []):
        if o["id"] in wanted:
            client.upsert_owner(o["id"], title=o["title"], contacts=_plain(o.get("contacts")) if "contacts" in o else None)
            print(f"  applied owner {o['title']}")
        for os_spec in o.get("ownerships", []):
            if os_spec["id"] not in wanted:
                continue
            spec = _plain(os_spec)
            if "dataproduct" in spec:
                spec["dataproduct"] = resolve_ref(spec["dataproduct"])
            sel = api.selection_to_proto(spec, resolve_ref=lambda v: v)
            client.upsert_ownership(o["id"], os_spec["id"], sel, api.alert_to_proto(spec.get("alert")))
            print(f"  applied ownership {(os_spec.get('query') or {}).get('name') or os_spec.get('dataproduct')}")
    print("Applied.")
    hints(("re-check drift", "governance-as-code plan -f <file>"),
          ("refresh file from API", "governance-as-code pull -f <file>"))


def do_prune(doc, client, assume_yes):
    owners = doc.get("owners", [])
    products = doc.get("dataproducts", [])
    present_owners = client.get_owners([o["id"] for o in owners if o.get("id")])
    present_products = client.get_products([p["id"] for p in products if p.get("id")])
    victims = [("owner", o["id"], o["title"]) for o in owners if o.get("id") in present_owners] + \
              [("dataproduct", p["id"], p["title"]) for p in products if p.get("id") in present_products]
    if not victims:
        print("Nothing to prune — none of the file's resources exist.")
        return
    for kind, _, title in victims:
        print(f"  - {kind:<11} {title}")
    confirm_or_die(f"Delete {len(victims)} resource(s)? (owners cascade to their ownerships)", assume_yes)
    for kind, i, title in victims:
        client.delete_owner(i) if kind == "owner" else client.delete_product(i)
        print(f"  deleted {kind} {title}")
    print("Pruned.")


# ---------------------------------------------------------------- export/pull

def _product_to_dict(p):
    d = {"id": p.id, "title": p.title}
    if p.folder:
        d["folder"] = p.folder
    prio = api.priority_from_proto(p.priority)
    if prio:
        d["priority"] = prio
    rql = api.rendered_resolver_ql(p)
    if rql:
        d["resolver_ql"] = rql
    return d


def _owner_to_dict(o, ownerships):
    d = {"id": o.id, "title": o.title}
    if o.contacts:
        d["contacts"] = [api.contact_from_proto(c) for c in o.contacts]
    if ownerships:
        d["ownerships"] = []
        for os_ in ownerships:
            entry = {"id": os_.id}
            sel = api.selection_from_proto(os_.selection)
            if "dataproduct_id" in sel:
                entry["dataproduct"] = sel["dataproduct_id"]
            else:
                entry["query"] = {k: v for k, v in sel["query"].items() if v}
            entry["alert"] = api.alert_from_proto(os_.alert)
            d["ownerships"].append(entry)
    return d


def do_export(client, out_path):
    import yaml
    products = [_product_to_dict(p) for p in client.list_products()]
    owners = [_owner_to_dict(o, client.list_ownerships(o.id)) for o in client.list_owners()]
    body = {}
    if products:
        body["dataproducts"] = products
    if owners:
        body["owners"] = owners
    header = (f"# yaml-language-server: $schema={SCHEMA_URL}\n"
              f"# Exported from Coalesce Quality by governance-as-code.\n")
    text = header + (yaml.safe_dump(body, sort_keys=False, allow_unicode=True) if body else "")
    if out_path and out_path != "-":
        with open(out_path, "w") as f:
            f.write(text)
        print(f"Exported {len(products)} product(s) and {len(owners)} owner(s) to {out_path}")
        hints(("review changes", f"governance-as-code plan -f {out_path}"))
    else:
        sys.stdout.write(text)


def do_pull(path, client):
    doc = load_doc(path)
    # Adopt-by-title so items authored without ids can still be refreshed.
    resolve_ids(doc, path, client, generate=False)
    n = 0
    remote_p = client.get_products([p["id"] for p in doc.get("dataproducts", []) if p.get("id")])
    for p in doc.get("dataproducts", []):
        rp = remote_p.get(p.get("id"))
        if not rp:
            continue
        p["title"] = rp.title
        if rp.folder:
            p["folder"] = rp.folder
        prio = api.priority_from_proto(rp.priority)
        if prio:
            p["priority"] = prio
        rql = api.rendered_resolver_ql(rp)
        if rql:
            p["resolver_ql"] = rql
        n += 1
    remote_o = client.get_owners([o["id"] for o in doc.get("owners", []) if o.get("id")])
    for o in doc.get("owners", []):
        ro = remote_o.get(o.get("id"))
        if not ro:
            continue
        o["title"] = ro.title
        if ro.contacts:
            o["contacts"] = [api.contact_from_proto(c) for c in ro.contacts]
        n += 1
    with open(path, "w") as f:
        _yaml_rt().dump(doc, f)
    print(f"Pulled {n} resource(s) into {path} (comments preserved).")


# ---------------------------------------------------------------- status

def do_status(args):
    exe = sys.argv[0]
    home = os.path.expanduser("~")
    if exe.startswith(home):
        exe = "~" + exe[len(home):]
    print(f"governance-as-code ({exe})")
    print("Declarative Coalesce Quality data products, owners & alert routing.\n")
    try:
        endpoint = resolve_endpoint(args)
        plugin = auth.resolve(endpoint, args.token, args.client_id, args.client_secret)
        client = api.Client(auth.channel(endpoint, plugin))
        print(f"endpoint : {endpoint}")
        print(f"auth     : {plugin.description}")
        print(f"products : {len(client.list_products())}")
        print(f"owners   : {len(client.list_owners())}")
    except Exception as e:
        print(f"not connected: {e}")
    hints(("preview a file", "governance-as-code plan -f governance.yaml"),
          ("apply a file", "governance-as-code apply -f governance.yaml --yes"),
          ("export workspace", "governance-as-code export -o governance.yaml"),
          ("sign in", "governance-as-code login"))


# ---------------------------------------------------------------- CLI plumbing

def hints(*pairs):
    print("\nhelp:")
    for desc, cmd in pairs:
        print(f"  {cmd:<52} # {desc}")


def do_completion(shell):
    """Print tab-completion setup (powered by argcomplete). Mirrors the intent of
    cobra's `completion` command in the idiomatic Python way."""
    prog = "governance-as-code"
    if shell == "fish":
        print(f"register-python-argcomplete --shell fish {prog} | source")
    elif shell == "zsh":
        print("autoload -U bashcompinit && bashcompinit")
        print(f'eval "$(register-python-argcomplete {prog})"')
    else:
        print(f'eval "$(register-python-argcomplete {prog})"')
    print(f"# Add the line(s) above to your shell rc. Requires `argcomplete` "
          f"(pip install argcomplete). One-time global setup: activate-global-python-argcomplete",
          file=sys.stderr)


def confirm_or_die(prompt, assume_yes):
    if assume_yes:
        return
    if not sys.stdin.isatty():
        raise CliError("refusing to proceed without confirmation in a non-interactive shell; pass --yes")
    if input(f"{prompt} [y/N] ").strip().lower() not in ("y", "yes"):
        raise CliError("aborted by user")


def _connect(args):
    endpoint = resolve_endpoint(args)
    plugin = auth.resolve(endpoint, args.token, args.client_id, args.client_secret)
    return api.Client(auth.channel(endpoint, plugin))


def build_parser():
    ap = argparse.ArgumentParser(prog="governance-as-code", description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--region", choices=sorted(set(REGIONS) | COMING_SOON), help="EU (default), US; APAC soon")
    ap.add_argument("--endpoint", help="override the API host directly")
    ap.add_argument("--token", help="bearer token (else QUALITY_TOKEN)")
    ap.add_argument("--client-id", help="OAuth2 client id (else QUALITY_CLIENT_ID)")
    ap.add_argument("--client-secret", help="OAuth2 client secret (else QUALITY_CLIENT_SECRET)")
    sub = ap.add_subparsers(dest="cmd")
    sub.add_parser("status", help="live workspace summary (default)")
    sub.add_parser("login", help="browser sign-in; cache a token for this tool")
    cp = sub.add_parser("completion", help="print shell completion setup")
    cp.add_argument("shell", nargs="?", choices=["bash", "zsh", "fish"], default="bash")
    for name in ("plan", "apply", "prune", "pull"):
        sp = sub.add_parser(name)
        sp.add_argument("-f", "--file", default="governance.yaml")
        if name in ("apply", "prune"):
            sp.add_argument("-y", "--yes", action="store_true", help="skip the confirmation prompt")
        if name in ("plan", "apply"):
            sp.add_argument("--json", action="store_true", help="print the plan as JSON")
    ep = sub.add_parser("export", help="read the workspace into a YAML file")
    ep.add_argument("-o", "--out", default="-", help="output file (default stdout)")
    return ap


def run(args):
    cmd = args.cmd or "status"
    if cmd == "status":
        do_status(args)
        return 0
    if cmd == "completion":
        do_completion(args.shell)
        return 0
    if cmd == "login":
        auth.login(resolve_endpoint(args))
        return 0
    if cmd == "export":
        do_export(_connect(args), args.out)
        return 0
    if cmd == "pull":
        do_pull(args.file, _connect(args))
        return 0

    doc = load_doc(args.file)
    client = _connect(args)
    resolve_ids(doc, args.file, client, generate=(cmd != "prune"))
    check_unique_ids(doc)  # re-check after adoption/generation
    actions = plan(doc, client)
    if getattr(args, "json", False):
        print(json.dumps({"actions": actions, "total": len(actions)}, indent=2))
    else:
        print_plan(actions)
    if cmd == "plan":
        if not getattr(args, "json", False):
            hints(("apply these", f"governance-as-code apply -f {args.file} --yes"))
        return 0
    if cmd == "apply":
        do_apply(doc, client, actions, args.yes)
    elif cmd == "prune":
        do_prune(doc, client, args.yes)
    return 0


def main():
    args = build_parser().parse_args()  # argparse exits 2 on unknown flags
    try:
        sys.exit(run(args))
    except CliError as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)
    except Exception as e:
        print(json.dumps({"error": f"{type(e).__name__}: {e}"}))
        sys.exit(1)


if __name__ == "__main__":
    main()
