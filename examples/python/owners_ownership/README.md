# Owners, Ownership & Data Products (Python)

This example manages **governance as code** with two new APIs:

- **`DataproductsService` (v2)** — data products: named, owned groupings of
  assets with a membership definition, priority and folder.
- **`OwnersService` (v1)** — owners and ownership: "alert routing as code". An
  owner is a responsible party with notification channels (contacts); an
  ownership assigns assets to an owner and configures the alerts routed to it.

Two runnable scripts (the Python counterparts of the `golang/owners_ownership`
example):

- **`dataproducts.py`** — the data-product maintenance lifecycle:
  create → set/patch definition (ResolverQL parts) → list resolved members →
  batch-get → partial update (with `etag`) → delete.
- **`owners.py`** — the owners lifecycle: create an owner with contacts → assign
  a data product to it (ownership + alert config) → add a query-based ownership →
  list → replace contacts (with `etag`) → delete. It creates a throwaway data
  product to own and cleans everything up at the end.

Both create real resources, exercise the API, and delete them again, so they are
safe to re-run.

> Looking for a declarative, file-driven way to manage these? See the
> **`governance-as-code`** tool under `proto_public/_repo/governance-as-code`.

## Prerequisites

Create client credentials at
[Coalesce Quality → API settings](https://app.synq.io/settings/api) with these
scopes and export them:

- `dataproducts.py`: **Read/Edit Data Products**.
- `owners.py`: **Read/Edit Owners**, **Read/Edit Ownership**, **Read/Edit Data
  Products**.

```bash
export SYNQ_CLIENT_ID=...
export SYNQ_CLIENT_SECRET=...
# Optional; defaults to the EU endpoint developer.synq.io. US: api.us.synq.io.
export API_ENDPOINT=developer.synq.io
```

## Setup & running

```bash
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt

python dataproducts.py
python owners.py
```

## Key API concepts

- **Caller-supplied UUID ids → idempotent writes.** You pass the `id` on create;
  repeating a request converges instead of duplicating.
- **Partial updates.** On `Upsert`, a field you set is written and a field you
  omit is left unchanged. Contacts / definitions use a presence wrapper: an
  omitted list means "leave unchanged", a present (possibly empty) list means
  "replace".
- **Optimistic concurrency via `etag`.** Read the `etag`, pass it back on
  update/delete to change only the version you last saw. A stale `etag` is
  rejected with `ABORTED` (HTTP 409). Omit it for last-write-wins.
- **`entity_id` is the cross-API reference.** Server-derived
  (`dataproduct-<uuid>`, `owner-<uuid>`); read it here and pass it to other APIs
  (lineage, entities, alerts). To assign a product to an owner, pass the bare
  `id`.
- **Ownership = routing as code.** An ownership's `AlertConfig` (severities,
  `notify_upstream`, ongoing strategy) is what the alerts API reports and filters
  on, attributed via the owner's `entity_id` and the ownership id.
- **Data products are leaves.** A data-product definition may not reference
  another data product or domain. An *ownership* query may.
