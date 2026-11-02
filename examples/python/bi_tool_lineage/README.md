# Modelling a BI tool end to end (Python)

Coalesce Quality has integrations for several BI tools. This example is about a
tool that has no integration (one you built, one you are trying out, or one
that is not supported yet). It shows how to put that tool in the catalog
yourself, with lineage that reaches down to the warehouse columns.

It uses [Metabase](https://www.metabase.com/) as the example, because most BI
tools have the same object model under different names. Nothing here is specific
to Metabase: rename the dataclasses in `metabase.py` and the mapping stays the
same.

This is the Python version of the [`golang/bi_tool_lineage`](../../golang/bi_tool_lineage)
example. Both write the same entities and print the same output.

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
        |   declared column lineage (there is no SQL)
        v
  Metabase dashboard             Commercial overview
        |
        |   a plain relationship (there are no columns)
        v
  Metabase subscription          Weekly commercial digest

  Metabase alert  ── checks ──>  Revenue by product category
```

There are four layers, and each hop uses a different mechanism on purpose.
Picking the right mechanism for each hop is the main skill. The rest is
bookkeeping.

As a result, a field on a dashboard that runs no SQL traces back to the physical
columns it came from:

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

Always try `SqlDefinition` first. A declaration is a snapshot of what was true
when you wrote it. A parse follows the query when it changes.

### The binding, in one paragraph

A `SqlDefinition` can only resolve names that have a warehouse address. A BI
model has no address (it is a saved query, not a table), so
`FROM order_items_enriched` matches nothing, is dropped, and the question ends up
with no upstream at all. There is no error: the empty result looks the same as a
query that really reads nothing. `SqlDefinition.references` binds the name, as
the SQL writes it, to the entity it stands for. This has three effects:

- A binding wins over the warehouse. If a real table has the same name, the
  binding still wins, so you can also use a binding as a manual override.
- Empty parts of a binding default from `database_context`, the same as an
  unqualified name in the SQL. So a bare `object_name` matches both
  `FROM my_model` and `FROM my_database.my_schema.my_model`.
- Names that are warehouse objects need no binding. The second question in this
  example reads a model and a raw table in one query, and binds only the model.

### Declare the schema, or lose column lineage without a warning

Every entity here declares its columns before anything reads them. This step is
easy to skip and expensive to skip. A `SELECT *` over a bound model expands only
when the model's columns are known, and a declared column edge only shows on a
column the entity has. If you skip it, the table-level edges still appear, but
the column-level edges are missing and nothing tells you.

## Running it

You need client credentials with these scopes: `Edit Entity Types`,
`Read Entity Types`, `Edit Entities`, `Edit Lineage`, `Read Lineage` and
`Edit Executions`. Create them on the API settings page of Coalesce Quality
([EU](https://app.synq.io/settings/api) · [US](https://app.us.synq.io/settings/api) · [AU](https://app.au.synq.io/settings/api)).

```bash
python -m venv venv && source venv/bin/activate
pip install -r requirements.txt

export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...

# The warehouse the BI tool queries. Its tables do not have to exist yet. An
# edge to an entity that Coalesce Quality has not seen is kept, and appears
# when the entity is ingested.
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

The script is idempotent: running it twice gives the same result. So you can
run it on a schedule. It ends by reading the lineage back. Keep that step when
you adapt the script. When something is wrong, the writes still succeed and the
graph is empty, and only the read-back shows the difference. Lineage is computed
asynchronously, so the read polls.

## The files

| File | What is in it |
|---|---|
| `auth.py` | The OAuth2 client-credentials token source and the gRPC auth plugin |
| `metabase.py` | The BI tool's own metadata, and the sample content. Read this first |
| `sync.py` | The mapping, the order the steps run in, and the read-back |

The metadata is declared inline, so the example needs only Coalesce Quality
credentials to run. A real integration fetches it from the tool instead.

## Adapting it

Decide these four things for your own tool. They are listed from the most
expensive to get wrong to the least:

1. Entity ids. Build them from the source system's stable id, never from a
   title. Everything is attached to the id. If you change an id, you get a
   second entity, and the first one is not renamed.
2. Type traits. `is_model`, `is_bi_like` and `is_test_type` decide how your
   entities rank, sort and behave everywhere in the product. Without them the
   catalog lists your entities but does not treat them as what they are.
3. One entity group that holds everything you own. The group makes deletion
   work: content deleted in the BI tool stops being sent, and the server
   deletes it. Without a group, you have to keep track of what you created last
   time, and that record will eventually be wrong. Send the full set every run.
   A partial send deletes everything it leaves out.
4. Check categories: leave them unset. Your workspace's categorisation rules
   set `category` and `governance_category`, and a value sent by a producer
   overrides those rules. Deriving a category from the tool's alert kind is the
   obvious thing to do, and it silently switches off every rule the workspace
   wrote. Send `package` and `kind`, which is what the tool says, and let the
   rules decide.

## Support

For questions about the API or about modelling your own tool, contact your
Technical Account Manager. See <https://docs.synq.io/support/support>.
