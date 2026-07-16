"""
Owners & ownership maintenance lifecycle — "alert routing as code" — using the
Coalesce Quality OwnersService (v1):

    create owner (+contacts) -> assign assets (ownership + alert config) ->
    list -> partial update (with etag) -> delete

An owner is a named responsible party with notification channels (contacts). An
ownership assigns a set of assets to an owner and configures the alerts routed
to it. Owner is the resource; ownership is its sub-resource — deleting an owner
deletes its ownerships.

Both ids are caller-supplied UUIDs, so every write is idempotent. Together they
replace clicking owners and alert rules together in the UI: define the
responsible party, its channels, what it owns, and how it should be alerted —
all as code.

To make the "assign a data product" path concrete, the example first creates a
throwaway data product to own, and deletes it at the end.

Prerequisites:
- SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET (scopes: Read/Edit Owners, Read/Edit
  Ownership, Read/Edit Data Products).
- Optionally API_ENDPOINT (defaults to developer.synq.io; US: api.us.synq.io).
"""

import os
import sys
import uuid

import grpc
from dotenv import load_dotenv

sys.path.insert(0, os.path.abspath(os.path.dirname(__file__)))

from auth import TokenSource, TokenAuth
from synq.dataproducts.v2 import (
    dataproducts_service_pb2 as dp_svc,
    dataproducts_service_pb2_grpc as dp_grpc,
    dataproduct_pb2 as dp,
    dataproduct_definition_pb2 as dp_def,
)
from synq.owners.v1 import (
    owners_service_pb2 as svc,
    owners_service_pb2_grpc as svc_grpc,
    ownership_pb2 as own,
    contact_pb2 as contact,
)
from synq.alerts.v1 import alerts_pb2
from synq.v1 import severity_pb2, pagination_pb2

load_dotenv()

CLIENT_ID = os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("SYNQ_CLIENT_SECRET")
API_ENDPOINT = os.getenv("API_ENDPOINT", "developer.synq.io")

if not CLIENT_ID or not CLIENT_SECRET:
    raise SystemExit("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set (scopes: Read/Edit Owners, Ownership, Data Products)")


def print_ownership(label, o):
    print(f"{label} id={o.id} etag={o.etag}")
    if o.selection.HasField("dataproduct_id"):
        print(f"  selection: data product {o.selection.dataproduct_id}")
    elif o.selection.HasField("query"):
        print(f'  selection: query "{o.selection.query.name}" -> {o.selection.query.rendered_resolver_ql}')
    a = o.alert
    sevs = [severity_pb2.Severity.Name(s) for s in a.severities]
    print(f"  alert: severities={sevs} notify_upstream={a.notify_upstream} disabled={a.is_disabled}\n")


def main():
    token_source = TokenSource(CLIENT_ID, CLIENT_SECRET, API_ENDPOINT)
    creds = grpc.composite_channel_credentials(
        grpc.ssl_channel_credentials(),
        grpc.metadata_call_credentials(TokenAuth(token_source)),
    )
    with grpc.secure_channel(
        f"{API_ENDPOINT}:443", creds, options=(("grpc.default_authority", API_ENDPOINT),)
    ) as channel:
        grpc.channel_ready_future(channel).result(timeout=10)
        print(f"Connected to {API_ENDPOINT}\n")
        owners = svc_grpc.OwnersServiceStub(channel)
        products = dp_grpc.DataproductsServiceStub(channel)

        # --- Setup: a data product to own ---
        print("=== Setup: create a data product to own ===")
        product_id = str(uuid.uuid4())
        products.Upsert(dp_svc.UpsertRequest(
            id=product_id,
            title="API Example — Owned Product",
            folder="API Examples",
            priority=dp.Dataproduct.PRIORITY_P1,
            definition=dp_def.DataproductDefinition(parts=[
                dp_def.DataproductDefinition.Part(
                    id=str(uuid.uuid4()),
                    query=dp_def.DataproductQuery(resolver_ql='with_type("table", filter=with_name("orders"))'),
                )
            ]),
        ))
        print(f"Created product {product_id}\n")

        # --- Step 1: Create an owner with notification channels ---
        # Contacts are the channels a fired alert is delivered to. Replace the
        # placeholders with real channels/addresses for your workspace.
        print("=== Step 1: Create owner (with contacts) ===")
        owner_id = str(uuid.uuid4())
        owner = owners.UpsertOwner(svc.UpsertOwnerRequest(
            id=owner_id,
            title="API Example — Data Platform",
            contacts=svc.ContactList(contacts=[
                contact.Contact(slack=contact.SlackChannelContact(channel="#data-alerts")),
                contact.Contact(email=contact.EmailContact(recipient_emails=["data-team@example.com"])),
            ]),
        )).owner
        print(f"Created owner id={owner.id}")
        # entity_id is server-derived (owner-<uuid>) — the value the alerts API
        # and other surfaces accept as an owner reference.
        print(f"entity_id={owner.entity_id} contacts={len(owner.contacts)} etag={owner.etag}\n")

        # --- Step 2: Assign the data product to the owner (ownership #1) ---
        # The ownership's AlertConfig is the "routing as code": which severities
        # fire, whether upstream issues count, and the repeat strategy.
        print("=== Step 2: Ownership #1 — own the data product, with alert routing ===")
        ownership_product_id = str(uuid.uuid4())
        up1 = owners.UpsertOwnership(svc.UpsertOwnershipRequest(
            owner_id=owner_id,
            id=ownership_product_id,
            selection=own.OwnershipSelection(dataproduct_id=product_id),
            alert=own.AlertConfig(
                severities=[severity_pb2.SEVERITY_ERROR, severity_pb2.SEVERITY_FATAL],
                notify_upstream=True,
                ongoing=alerts_pb2.OngoingAlertsStrategy(disabled=alerts_pb2.OngoingAlertsStrategy.Disabled()),
            ),
        )).ownership
        print_ownership("ownership #1", up1)

        # --- Step 3: A second ownership selecting assets by query (ownership #2) ---
        # Unlike a data-product definition (a leaf), an ownership query MAY
        # reference data products and domains.
        print("=== Step 3: Ownership #2 — select by ResolverQL, weekly digest ===")
        ownership_query_id = str(uuid.uuid4())
        up2 = owners.UpsertOwnership(svc.UpsertOwnershipRequest(
            owner_id=owner_id,
            id=ownership_query_id,
            selection=own.OwnershipSelection(
                query=own.OwnershipQuery(name="Critical models", resolver_ql='with_type("model", filter=with_name("revenue"))'),
            ),
            alert=own.AlertConfig(
                severities=[severity_pb2.SEVERITY_FATAL],
                ongoing=alerts_pb2.OngoingAlertsStrategy(
                    schedule=alerts_pb2.OngoingAlertsStrategy.Schedule(cron="0 9 * * MON"),
                ),
            ),
        )).ownership
        print_ownership("ownership #2", up2)

        # --- Step 4: List the owner's ownerships ---
        print("=== Step 4: List ownerships ===")
        listed = owners.ListOwnerships(svc.ListOwnershipsRequest(owner_id=owner_id, pagination=pagination_pb2.Pagination()))
        print(f"Owner has {len(listed.ownerships)} ownership(s)\n")

        # --- Step 5: Partial owner update with optimistic concurrency ---
        # Contacts are replace-semantics behind a presence wrapper: omit `contacts`
        # to leave them unchanged, or send the full desired set to replace them.
        print("=== Step 5: Replace contacts (add MS Teams) with etag ===")
        stale_etag = owner.etag
        owner = owners.UpsertOwner(svc.UpsertOwnerRequest(
            id=owner_id,
            etag=owner.etag,
            contacts=svc.ContactList(contacts=[
                contact.Contact(slack=contact.SlackChannelContact(channel="#data-alerts")),
                contact.Contact(email=contact.EmailContact(recipient_emails=["data-team@example.com"])),
                contact.Contact(ms_teams=contact.MsTeamsContact(channel_id="19:example-teams-channel-id@thread.tacv2")),
            ]),
        )).owner
        print(f"Updated owner now has {len(owner.contacts)} contacts, new etag={owner.etag}")

        try:
            owners.UpsertOwner(svc.UpsertOwnerRequest(id=owner_id, title="should be rejected", etag=stale_etag))
            print("Unexpected: stale etag accepted\n")
        except grpc.RpcError as e:
            ok = e.code() == grpc.StatusCode.ABORTED
            print(f"Stale etag rejected with {e.code().name} ({'expected' if ok else 'unexpected'})\n")

        # --- How this ties to the alerts API ---
        print("=== Reference: how alerts point back here ===")
        print(f"owner entity_id : {owner.entity_id}")
        print(f"ownership ids   : {ownership_product_id}, {ownership_query_id}\n")

        # --- Cleanup ---
        print("=== Cleanup ===")
        owners.DeleteOwnership(svc.DeleteOwnershipRequest(id=ownership_query_id))
        owners.DeleteOwner(svc.DeleteOwnerRequest(id=owner_id, purge=True))
        products.Delete(dp_svc.DeleteRequest(id=product_id, purge=True))
        print(f"Deleted ownerships, owner {owner_id}, and product {product_id}")

        print("\nDone: owners & ownership lifecycle exercised end to end.")


if __name__ == "__main__":
    main()
