# Owners, ownership and data products (Python)

This example manages governance as code with two APIs:

- `DataproductsService` (v2) manages data products. A data product is a named,
  owned group of assets with a membership definition, a priority and a folder.
- `OwnersService` (v1) manages owners and ownership ("alert routing as code").
  An owner is a responsible party with notification channels (contacts). An
  ownership assigns assets to an owner and configures the alerts sent to it.

There are two scripts. They are the Python versions of the
[`golang/owners_ownership`](../../golang/owners_ownership) example.

- `dataproducts.py` runs the data-product lifecycle: create, set and change the
  definition (ResolverQL parts), list the resolved members, batch-get, partial
  update (with `etag`), delete.
- `owners.py` runs the owners lifecycle: create an owner with contacts, assign a
  data product to it (ownership and alert config), add a query-based ownership,
  list, replace the contacts (with `etag`), delete. It creates a temporary data
  product to own, and deletes everything at the end.

Both scripts create real resources, call the API, and delete the resources at
the end, so you can run them again safely.

> To manage these resources from files instead, see the
> [`governance-as-code`](../../../governance-as-code) tool.

## Prerequisites

Create client credentials in Coalesce Quality → API settings ([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)) with these
scopes, and export them:

- `dataproducts.py`: Read Data Products and Edit Data Products.
- `owners.py`: Read Owners, Edit Owners, Read Ownership, Edit Ownership, Read
  Data Products and Edit Data Products.

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
# Optional. The default is the EU endpoint, developer.synq.io.
# Other regions: api.us.synq.io for US, api.au.synq.io for AU.
export QUALITY_API_ENDPOINT=developer.synq.io
```

## Setup and running

```bash
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt

python dataproducts.py
python owners.py
```

## API concepts

- The caller supplies the ids (UUIDs), so writes are idempotent. You pass the
  `id` on create, and if you repeat a request, it updates the same resource and
  does not create a copy.
- Partial updates. On `Upsert`, a field you set is written and a field you omit
  is left unchanged. Contacts and definitions use a presence wrapper: if you omit
  the list, it is left unchanged. If you send a list (even an empty one), it
  replaces the stored list.
- Optimistic concurrency with `etag`. Read the `etag` and pass it back on update
  or delete, so you only change the version you last read. A stale `etag` is
  rejected with `ABORTED` (HTTP 409). Omit it if the last write should win.
- `entity_id` is the reference you use in other APIs. The server derives it
  (`dataproduct-<uuid>`, `owner-<uuid>`). Read it here and pass it to other APIs
  (lineage, entities, alerts). To assign a data product to an owner, pass either
  the bare `id` or the `entity_id`.
- An ownership is alert routing as code. The alerts API reports and filters on
  an ownership's `AlertConfig` (severities, `notify_upstream`, ongoing
  strategy). An alert names its owner in `Owner.owner_path`, which takes the
  owner's `entity_id` (`owner-<uuid>`, not the bare `id`), and the ownership in
  `Owner.ownership_id`.
- A data-product definition cannot reference another data product or a domain.
  An ownership query can.
