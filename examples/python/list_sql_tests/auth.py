"""
Contains helper classes for authorization to the SYNQ server.
"""

import requests
import time
import grpc


class TokenAuth(grpc.AuthMetadataPlugin):
    """AuthMetadataPlugin which adds the access token to the outgoing context metadata."""

    def __init__(self, token_source):
        self._token_source = token_source

    def __call__(self, context, callback):
        try:
            token = self._token_source.get_token()
            callback([("authorization", f"Bearer {token}")], None)
        except Exception as e:
            callback(None, e)


class TokenSource:
    """Token source which maintains the access token and refreshes it when it is expired."""

    def __init__(self, client_id, client_secret, api_endpoint):
        self.api_endpoint = api_endpoint
        self.token_url = f"https://{self.api_endpoint}/oauth2/token"
        self.client_id = client_id
        self.client_secret = client_secret
        self.token = self.obtain_token()

    def obtain_token(self):
        resp = requests.post(
            self.token_url,
            data={
                "client_id": self.client_id,
                "client_secret": self.client_secret,
                "grant_type": "client_credentials",
            },
        )
        return resp.json()

    def get_token(self) -> str:
        expires_at = self.token["expires_in"] + time.time()
        is_expired = time.time() > expires_at
        if is_expired:
            self.token = self.obtain_token()
        return self.token["access_token"]
