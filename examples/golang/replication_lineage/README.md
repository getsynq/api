# Lineage across a replication tool

A replication tool (Airbyte, Fivetran, a CDC pipeline, your own loader) copies a
table from one database into a table in another warehouse. Coalesce Quality
already has both tables; what it does not have is the hop between them. This
example declares it, with column lineage, and reads it back.

```
  Postgres table        app_db.public.orders
        |
        |   INSERT INTO target_table (...) SELECT ... FROM source_table
        v
  replication stream    custom entity, one per replicated table
        |
        v
  Snowflake table       RAW.APP_DB.ORDERS
```

## Why a custom entity in the middle

A relationship between two tables Coalesce Quality already knows is refused: a
relationship needs a custom entity on at least one side. Model the thing that
copies the rows, because it is what fails, lags or drops a column, and it is where
you want an incident to land.

Declare **one entity per replicated table**. A single entity for the whole
connection would make every source table read as upstream of every target table
the connection writes.

## Why SQL, and why bindings

The hop is a `SqlDefinition` on the stream entity. SQL says which source column
lands in which target column, including renames, and the parser classifies each
one as a passthrough, a rename or a transformation.

The SQL is never executed, only parsed. Its two table names are placeholders, and
each is **bound** to the real table with a `SqlTableReference`. That is what lets
the source and the target sit on different platforms: a bound name resolves to
its entity directly, instead of being looked up by warehouse address on the
SQL's own platform.

Write an explicit column list on the `INSERT`. It pairs each selected column
with its target column; without it the pairing is positional.

## Running it

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...
# export QUALITY_API_ENDPOINT=api.us.synq.io   # US workspaces; api.au.synq.io for AU
go run .
```

The client needs `SCOPE_ENTITY_EDIT`, `SCOPE_ENTITY_TYPE_EDIT` and
`SCOPE_LINEAGE_READ`.

Edit `streams` in `main.go` to point at tables that exist in your workspace. The
identifiers must match how the tables are already known: an identifier that
matches nothing is accepted, and the edge simply never appears. The verify step
at the end is there to catch that.

Re-running is safe. The entities are in one group, so a stream you remove from
`streams` is deleted on the next run.
