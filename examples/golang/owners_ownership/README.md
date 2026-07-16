# Owners, Ownership & Data Products (Go)

This example manages **governance as code** with two new APIs:

- **`DataproductsService` (v2)** — data products: named, owned groupings of
  assets with a membership definition, priority and folder.
- **`OwnersService` (v1)** — owners and ownership: "alert routing as code". An
  owner is a responsible party with notification channels (contacts); an
  ownership assigns assets to an owner and configures the alerts routed to it.

Two runnable programs:

- **`dataproducts/`** — the data-product maintenance lifecycle:
  create → set/patch definition (ResolverQL parts) → list resolved members →
  batch-get → partial update (with `etag`) → delete.
- **`owners/`** — the owners lifecycle: create an owner with contacts → assign a
  data product to it (ownership + alert config) → add a query-based ownership →
  list → replace contacts (with `etag`) → delete. It creates a throwaway data
  product to own and cleans everything up at the end.

Both create real resources, exercise the API, and delete them again, so they are
safe to re-run.

## Prerequisites

Create client credentials at
[Coalesce Quality → API settings](https://app.synq.io/settings/api) with these
scopes and export them:

- `dataproducts/`: **Read/Edit Data Products**.
- `owners/`: **Read/Edit Owners**, **Read/Edit Ownership**, **Read/Edit Data
  Products**.

```bash
export SYNQ_CLIENT_ID=...
export SYNQ_CLIENT_SECRET=...
# Optional; defaults to the EU endpoint developer.synq.io.
# For the US region use api.us.synq.io.
export API_ENDPOINT=developer.synq.io
```

## Running

```bash
go run ./dataproducts
go run ./owners
```

## Authoring selections with ResolverQL

Membership (data products) and asset selections (ownerships) are authored in
**ResolverQL**, a compact text query language — e.g.
`with_type("table", filter=with_name("orders"))` or
`in_product(["dataproduct-<uuid>"])`. The server compiles and stores the query
canonically and echoes it back on reads as `rendered_resolver_ql` (so
`with_type("table")` comes back expanded to every warehouse table type). Copy
real paths and ids from the Synq UI; the examples use placeholder queries that
may match nothing in your workspace (an empty member list is then normal).

## Key API concepts

- **Caller-supplied UUID ids → idempotent writes.** You pass the `id` on create.
  Repeating a request converges to the same resource instead of duplicating, so
  a ret&#8209;ry or a re&#8209;run of your config is safe.
- **Partial updates.** On `Upsert`, a field you set is written and a field you
  omit is left unchanged. Contacts / definitions use a presence wrapper so an
  omitted list means "leave unchanged" and a present (possibly empty) list means
  "replace".
- **Optimistic concurrency via `etag`.** Read the `etag`, pass it back on
  update/delete to change only the version you last saw. A stale `etag` is
  rejected with `ABORTED` (HTTP 409). Omit it for last-write-wins.
- **`entity_id` is the cross-API reference.** Server-derived (`dataproduct-<uuid>`,
  `owner-<uuid>`); read it here and pass it to other APIs (lineage, ownership,
  alerts) to point at the same resource — never construct it by hand.
- **Ownership = routing as code.** An ownership's `AlertConfig` (severities,
  `notify_upstream`, ongoing strategy) is what the alerts API reports and filters
  on, attributed via the owner's `entity_id` and the ownership id.
- **Data products are leaves.** A data-product definition may not reference
  another data product or domain. An *ownership* query may — routing alerts for a
  whole product or domain is a first-class use.
