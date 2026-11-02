# Owners, ownership and data products (Go)

This example manages governance as code with two APIs:

- `DataproductsService` (v2) manages data products. A data product is a named,
  owned group of assets, with a membership definition, a priority and a folder.
- `OwnersService` (v1) manages owners and ownership, which is alert routing as
  code. An owner is a responsible party with notification channels (contacts).
  An ownership assigns assets to an owner and configures the alerts sent to it.

There are two programs:

- `dataproducts/` runs the data product lifecycle: create, set and change the
  definition (ResolverQL parts), list the resolved members, batch get, partial
  update (with `etag`) and delete.
- `owners/` runs the owners lifecycle: create an owner with contacts, assign a
  data product to it (ownership and alert config), add an ownership based on a
  query, list, replace the contacts (with `etag`) and delete. It creates a
  temporary data product to own and deletes everything at the end.

Both programs create real resources, call the API, and delete the resources
again. You can run them again safely.

## Prerequisites

Create client credentials in Coalesce Quality, on the API settings page
([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)), with these
scopes, and export them:

- `dataproducts/`: Read Data Products and Edit Data Products.
- `owners/`: Read Owners, Edit Owners, Read Ownership, Edit Ownership, Read Data
  Products and Edit Data Products.

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
# Optional. The default is the EU endpoint, developer.synq.io.
# For other regions: api.us.synq.io for US, api.au.synq.io for AU.
export QUALITY_API_ENDPOINT=developer.synq.io
```

## Running

```bash
go run ./dataproducts
go run ./owners
```

## Writing selections in ResolverQL

You write data product membership and ownership asset selections in ResolverQL,
a short text query language. For example:
`with_type("table", filter=with_name("orders"))` or
`in_product(["dataproduct-<uuid>"])`. The server compiles the query, stores it in
canonical form, and returns it on reads as `rendered_resolver_ql`. For example,
`with_type("table")` comes back expanded to every warehouse table type. Copy
real paths and IDs from the Coalesce Quality app. The examples use placeholder
queries that can match nothing in your workspace. An empty member list is then
normal.

## API concepts

- You choose the UUID `id`, so writes are idempotent. You send the `id` on
  create. If you repeat a request, it updates the same resource and does not
  create a duplicate. So you can safely retry a request or run your
  configuration again.
- Updates are partial. On `Upsert`, a field you set is written and a field you
  omit stays unchanged. Contacts and definitions use a wrapper with presence: if
  you omit the list, it stays unchanged. If you send the list (even an empty
  one), it replaces the stored list.
- `etag` gives optimistic concurrency. Read the `etag` and send it back on update
  or delete, so the change applies only to the version you read. A stale `etag`
  is rejected with `ABORTED` (HTTP 409). Omit the `etag` and the last write wins.
- `entity_id` is the reference that other APIs use. The server creates it
  (`dataproduct-<uuid>`, `owner-<uuid>`). Read it here and give it to other APIs
  (lineage, ownership, alerts) to point at the same resource. Do not build it
  yourself.
- An ownership routes alerts. Its `AlertConfig` (severities, `notify_upstream`,
  ongoing strategy) is what the alerts API reports and filters on. An alert names
  its owner in `Owner.owner_path`, which takes the owner's `entity_id`
  (`owner-<uuid>`, not the bare `id`), and the ownership in `Owner.ownership_id`.
- A data product cannot point to other data products. Its definition cannot
  reference another data product or a domain. An ownership query can, so you can
  route the alerts of a whole data product or domain to one owner.
