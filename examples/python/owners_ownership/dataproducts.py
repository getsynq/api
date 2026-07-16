"""
Data-products maintenance lifecycle using the Coalesce Quality
DataproductsService (v2):

    upsert -> set-definition -> add/remove parts -> list-members -> batch-get ->
    partial update (with etag) -> delete

A data product is a named, owned grouping of assets with a membership
definition. The id is a caller-supplied UUID, so every write is idempotent —
re-running this example converges instead of duplicating.

The membership definition is authored here in ResolverQL, the compact text
query language. The server compiles and stores it canonically and echoes it
back on reads as `rendered_resolver_ql`.

Prerequisites:
- SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET (scopes: Read/Edit Data Products).
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
    dataproducts_service_pb2 as svc,
    dataproducts_service_pb2_grpc as svc_grpc,
    dataproduct_pb2 as dp,
    dataproduct_definition_pb2 as dp_def,
)
from synq.v1 import pagination_pb2

load_dotenv()

CLIENT_ID = os.getenv("SYNQ_CLIENT_ID")
CLIENT_SECRET = os.getenv("SYNQ_CLIENT_SECRET")
API_ENDPOINT = os.getenv("API_ENDPOINT", "developer.synq.io")

if not CLIENT_ID or not CLIENT_SECRET:
    raise SystemExit("SYNQ_CLIENT_ID and SYNQ_CLIENT_SECRET must be set (scopes: Read/Edit Data Products)")


def print_definition(product):
    parts = product.definition.parts
    print(f"definition has {len(parts)} part(s), etag={product.etag}")
    for p in parts:
        if p.HasField("query"):
            print(f"  - part {p.id}: {p.query.rendered_resolver_ql}")
        elif p.entity_id:
            print(f"  - part {p.id}: entity_id={p.entity_id}")
    print()


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
        stub = svc_grpc.DataproductsServiceStub(channel)

        # The caller owns the id. A fresh UUID creates a new product; re-using a
        # known id updates that product in place.
        product_id = str(uuid.uuid4())

        # --- Step 1: Create ---
        # Only `title` is required on create. `folder` and `priority` are optional.
        print("=== Step 1: Create data product ===")
        created = stub.Upsert(svc.UpsertRequest(
            id=product_id,
            title="API Example — Orders",
            description="Created by the owners_ownership Python example.",
            folder="API Examples",
            priority=dp.Dataproduct.PRIORITY_P1,
        )).dataproduct
        print(f"Created id={created.id}")
        # entity_id is server-derived (dataproduct-<uuid>) — the value other APIs
        # (lineage, entities, alerts) accept as a reference to this product.
        print(f"entity_id={created.entity_id} (use this to reference the product elsewhere)")
        print(f"etag={created.etag} priority={dp.Dataproduct.Priority.Name(created.priority)}\n")

        # --- Step 2: Replace the whole definition (SetDefinition) ---
        # A definition is a list of parts OR'd together. Here a single ResolverQL
        # query part. Replace the placeholder query with one matching your workspace.
        print("=== Step 2: Set definition (ResolverQL) ===")
        set_resp = stub.SetDefinition(svc.SetDefinitionRequest(
            id=product_id,
            etag=created.etag,
            definition=dp_def.DataproductDefinition(parts=[
                dp_def.DataproductDefinition.Part(
                    id=str(uuid.uuid4()),
                    query=dp_def.DataproductQuery(resolver_ql='with_type("table", filter=with_name("orders"))'),
                )
            ]),
        )).dataproduct
        print_definition(set_resp)
        product = set_resp

        # --- Step 3: Add a second part (UpsertDefinitionPart) ---
        # Parts have caller-supplied ids so an upsert is idempotent. Definition
        # writes bump the etag, so pass the latest one.
        print("=== Step 3: Add a second part ===")
        part_id = str(uuid.uuid4())
        product = stub.UpsertDefinitionPart(svc.UpsertDefinitionPartRequest(
            id=product_id,
            etag=product.etag,
            part=dp_def.DataproductDefinition.Part(
                id=part_id,
                query=dp_def.DataproductQuery(resolver_ql='with_type("view", filter=with_name("orders"))'),
            ),
        )).dataproduct
        print_definition(product)

        # --- Step 4: Remove that part (RemoveDefinitionPart) ---
        print("=== Step 4: Remove the part ===")
        product = stub.RemoveDefinitionPart(svc.RemoveDefinitionPartRequest(
            id=product_id, part_id=part_id, etag=product.etag,
        )).dataproduct
        print_definition(product)

        # --- Step 5: List resolved members ---
        # The definition resolves to concrete assets (opaque entity ids). Empty is
        # normal if the placeholder query matches nothing in your workspace.
        print("=== Step 5: List members ===")
        members = stub.ListMembers(svc.ListMembersRequest(id=product_id, pagination=pagination_pb2.Pagination()))
        print(f"Resolved {len(members.entity_ids)} member(s)")
        for e in members.entity_ids[:5]:
            print(f"  - {e}")
        if len(members.entity_ids) > 5:
            print("  …")
        print()

        # --- Step 6: BatchGet (exclude the definition for a lighter read) ---
        print("=== Step 6: BatchGet ===")
        got = stub.BatchGet(svc.BatchGetRequest(ids=[product_id], exclude_definition=True)).dataproducts[product_id]
        print(f'title="{got.title}" folder="{got.folder}" '
              f"priority={dp.Dataproduct.Priority.Name(got.priority)} "
              f"(definition excluded: {len(got.definition.parts)} parts)\n")

        # --- Step 7: Partial update with optimistic concurrency ---
        # Upsert only the fields you set; omitted fields are left unchanged. Pass
        # the etag you last read so a concurrent edit is not silently overwritten.
        print("=== Step 7: Partial update (title only) with etag ===")
        stale_etag = product.etag
        product = stub.Upsert(svc.UpsertRequest(
            id=product_id,
            title="API Example — Orders (P2)",
            priority=dp.Dataproduct.PRIORITY_P2,
            etag=product.etag,
        )).dataproduct
        print(f'Updated title="{product.title}" '
              f"priority={dp.Dataproduct.Priority.Name(product.priority)} new etag={product.etag}")

        # Re-using the now-stale etag is rejected — the concurrency guard.
        try:
            stub.Upsert(svc.UpsertRequest(id=product_id, title="should be rejected", etag=stale_etag))
            print("Unexpected: stale etag accepted\n")
        except grpc.RpcError as e:
            ok = e.code() == grpc.StatusCode.ABORTED
            print(f"Stale etag rejected with {e.code().name} ({'expected' if ok else 'unexpected'})\n")

        # --- Step 8: Delete (purge to release the id) ---
        print("=== Step 8: Delete ===")
        stub.Delete(svc.DeleteRequest(id=product_id, purge=True, etag=product.etag))
        after = stub.BatchGet(svc.BatchGetRequest(ids=[product_id]))
        print(f"Deleted {product_id}; BatchGet after delete returned {len(after.dataproducts)} product(s)")

        print("\nDone: data-product maintenance lifecycle exercised end to end.")


if __name__ == "__main__":
    main()
