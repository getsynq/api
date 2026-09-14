# Modelling a BI tool end to end (Python)

Coalesce Quality has integrations for several BI tools. This example is about the
other case: a tool there is no integration for — one you built, one you are
piloting, or one nobody has got to yet — and how to put it in the catalog
yourself, with lineage that reaches all the way down to the warehouse columns.

It uses [Metabase](https://www.metabase.com/) as a concrete stand-in, because its
object model is the one most BI tools have under different names. Nothing here is
Metabase-specific: rename the dataclasses in `metabase.py` and the mapping is
unchanged.

This is the Python counterpart of the `golang/bi_tool_lineage` example. The two
write the same estate and print the same thing.

## What it builds

```
  BigQuery tables                order_items, products, customers
        |
        |   SQL, resolved by warehouse address
        v
  Metabase models                Order items enriched, Customer regions
        |
        |   SQL, resolved by a BINDING
        v
  Metabase questions             Revenue by product category, Customers by region
        |
        |   declared column lineage — no SQL exists
        v
  Metabase dashboard             Commercial overview
        |
        |   a plain relationship — no columns exist
        v
  Metabase subscription          Weekly commercial digest

  Metabase alert  ── checks ──>  Revenue by product category
```

Four layers, and **each hop uses a different mechanism on purpose**. Picking the
right one per hop is the whole skill; the rest is bookkeeping.

The result is that a field on a dashboard that runs no SQL at all traces back to
the physical columns it came from:

```
dashboard.revenue_by_category.revenue
  <- question.revenue
       <- model.net_revenue
            <- order_items.quantity
            <- order_items.unit_price
            <- order_items.discount
```

## Which mechanism, and when

| You have | Use | Why |
|---|---|---|
| SQL, over real warehouse tables | `SqlDefinition` | Both grains are derived from the SQL, and stay correct when the SQL changes |
| SQL, naming something the warehouse cannot address | `SqlDefinition` + `references` | The binding says what the name means; everything else is derived as usual |
| No SQL, but you know the column mapping | `ColumnLineage` | Declare the answer instead of deriving one |
| No SQL and no columns | `Relationship` | The edge is real; there is nothing to say at column grain |
| Something that validates another entity | `CheckCategory` + a check relationship | A check is not a stage data flows through |

The one to reach for first is always `SqlDefinition`. A declaration is a snapshot
of what was true when it was written; a parse follows the query.

### The binding, in one paragraph

A `SqlDefinition` can only resolve names that live at a warehouse address. A BI
model has no address — it is a saved query, not a table — so
`FROM order_items_enriched` matches nothing, is dropped, and the question ends up
with no upstream at all. There is no error: an empty result looks exactly like a
query that genuinely reads nothing. `SqlDefinition.references` binds the name, as
the SQL writes it, to the entity it stands for. Three things follow:

- **A binding beats the warehouse.** If a real table shared the name, the binding
  still wins, which is what makes it usable as a manual override.
- **Empty parts default from `database_context`**, exactly as an unqualified name
  in the SQL does — so a bare `object_name` matches both `FROM my_model` and
  `FROM my_database.my_schema.my_model`.
- **Names that ARE warehouse objects need no binding.** The second question in
  this example reads a model and a raw table in one query, and binds only the
  model.

### Declare the schema, or lose the column grain silently

Every entity here declares its columns before anything reads them. This is the
step most easily skipped and the most expensive to skip: a `SELECT *` over a
bound model expands only over a model whose columns are known, and a declared
column edge only shows against a column the entity actually has. Skip it and the
table-level edges still appear — only the column-level ones are missing, with
nothing saying so.

## Running it

You need client credentials with `Manage entity types`, `Manage entities`,
`Manage lineage` and `Manage executions`. Create them on the API settings page of
your Coalesce Quality app.

```bash
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt

export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...

# The warehouse the BI tool queries. Its tables do not have to exist yet — an
# edge to an entity Coalesce Quality has not seen is remembered, and appears
# once it is ingested.
export BIGQUERY_PROJECT=my-gcp-project
export BIGQUERY_DATASET=analytics

# Optional: the repository a serialized export of the BI content is committed
# to, so Coalesce Quality can read its history from the commits.
export BI_EXPORT_GIT_REPO=git@github.com:my-org/my-bi-export.git
export BI_EXPORT_GIT_BRANCH=main

python sync.py
```

The default endpoint is `developer.synq.io` (EU). Set `QUALITY_API_ENDPOINT` to
`api.us.synq.io` for US or `api.au.synq.io` for AU.

The script is idempotent — running it twice produces the same estate — so it is a
cron job, not a migration. It ends by reading the lineage back, which is worth
keeping when you adapt it: every way this goes wrong produces a valid-looking
write and an empty graph, and reading back is the only thing that tells the two
apart. Lineage is computed asynchronously, so the read polls.

## The files

| File | What is in it |
|---|---|
| `auth.py` | The OAuth2 client-credentials token source and the gRPC auth plugin |
| `metabase.py` | The BI tool's own metadata, and the sample content. Read this first |
| `sync.py` | The mapping, the order the steps run in, and the read-back |

The metadata is declared inline so the example runs with nothing but Coalesce
Quality credentials. A real integration fetches it from the tool instead — see
the [`integrations_management`](../integrations_management) example for that
half.

## Adapting it

Four things to decide for your own tool, in order of how expensive they are to
get wrong:

1. **Entity ids.** Build them from the source system's stable id, never from a
   title. Everything hangs off the id, and changing one creates a second entity
   rather than renaming the first.
2. **Type traits.** `is_model`, `is_bi_like` and `is_test_type` decide how your
   entities rank, sort and behave platform-wide. They are the difference between
   a catalog that understands your tool and one that merely lists it.
3. **One entity group, holding everything you own.** It is what makes deletion
   work: content deleted in the BI tool stops being sent, and the server deletes
   it. Without it you have to remember on your own side what you created last
   time, and that state will eventually disagree with reality. Send the full set
   every run — a partial send is a deletion of everything omitted.
4. **Check categories: leave them unset.** `category` and `governance_category`
   are resolved by your workspace's own categorisation rules, and a value sent by
   a producer outranks those rules. Deriving one from the tool's alert kind —
   the obvious thing to do — silently switches off every rule the workspace
   wrote. Send `package` and `kind`, which is what the tool actually says, and
   let the rules decide.

## Support

Questions about the API or about modelling a tool of your own go to your
Technical Account Manager — see <https://docs.synq.io/support/support>.
