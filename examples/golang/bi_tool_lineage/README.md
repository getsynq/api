# Modelling a BI tool end to end

Coalesce Quality has integrations for several BI tools. This example is for a
tool that has no integration (one you built, one you are trying out, or one
that is not supported yet). It shows how to add that tool to the catalog
yourself, with lineage that goes all the way down to the warehouse columns.

It uses [Metabase](https://www.metabase.com/) as a concrete example, because most
BI tools have the same object model under different names. Nothing here is
specific to Metabase: rename the structs in `metabase.go` and the mapping stays
the same.

There is a Python version of this example under `python/bi_tool_lineage`. Both
versions write the same entities and print the same output.

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
        |   declared column lineage, no SQL exists
        v
  Metabase dashboard             Commercial overview
        |
        |   a plain relationship, no columns exist
        v
  Metabase subscription          Weekly commercial digest

  Metabase alert  ── checks ──>  Revenue by product category
```

There are four layers, and each hop uses a different mechanism on purpose. The
main skill is to pick the right mechanism for each hop.

As a result, a field on a dashboard that runs no SQL traces back to the
physical columns it came from:

```
dashboard.revenue_by_category.revenue
  <- question.revenue
       <- model.net_revenue
            <- order_items.quantity
            <- order_items.unit_price
            <- order_items.discount
```

## Which mechanism to use

| You have | Use | Why |
|---|---|---|
| SQL over real warehouse tables | `SqlDefinition` | Table and column lineage come from the SQL, and stay correct when the SQL changes |
| SQL that names something the warehouse cannot address | `SqlDefinition` + `references` | The binding says what the name means. Everything else comes from the SQL as usual |
| No SQL, but you know the column mapping | `ColumnLineage` | You declare the result directly |
| No SQL and no columns | `Relationship` | The edge is real, but there is nothing to say about columns |
| Something that validates another entity | `CheckCategory` + a check relationship | A check is not a step that data flows through |

Always try `SqlDefinition` first. A declaration shows what was true when you
wrote it. A parsed definition follows the query when it changes.

### The binding

A `SqlDefinition` can only resolve names that have a warehouse address. A BI
model has no address, because it is a saved query and not a table. So
`FROM order_items_enriched` matches nothing and is dropped, and the question
gets no upstream at all. You get no error: the empty result looks the same as a
query that reads nothing. `SqlDefinition.references` binds the name, as the SQL
writes it, to the entity that it stands for. This has three effects:

- A binding has priority over the warehouse. If a real table has the same name,
  the binding still wins. This is why you can also use a binding as a manual
  override.
- Empty parts of a binding default from `database_context`, in the same way as
  an unqualified name in the SQL. So a bare `object_name` matches both
  `FROM my_model` and `FROM my_database.my_schema.my_model`.
- Names that are warehouse objects need no binding. The second question in
  this example reads a model and a raw table in one query, and binds only the
  model.

### Declare the schema, or you lose column lineage without a warning

Every entity in this example declares its columns before anything reads them.
This step is easy to skip, and skipping it has a high cost. A `SELECT *` over a
bound model expands only when the model's columns are known, and a declared
column edge shows only for a column that the entity has. If you skip it, the
table-level edges still appear, but the column-level edges are missing and
nothing tells you.

## Running it

You need client credentials with these scopes: `Edit Entity Types`,
`Read Entity Types`, `Edit Entities`, `Edit Lineage`, `Read Lineage` and
`Edit Executions`. Create them on the API settings page of your Coalesce Quality
app.

```bash
export QUALITY_CLIENT_ID=...
export QUALITY_CLIENT_SECRET=...

# The warehouse the BI tool queries. Its tables do not have to exist yet. An
# edge to an entity that Coalesce Quality has not seen is kept, and it appears
# when the entity is ingested.
export BIGQUERY_PROJECT=my-gcp-project
export BIGQUERY_DATASET=analytics

# Optional: the repository where a serialized export of the BI content is
# committed, so Coalesce Quality can read its history from the commits.
export BI_EXPORT_GIT_REPO=git@github.com:my-org/my-bi-export.git
export BI_EXPORT_GIT_BRANCH=main

go run .
```

The default endpoint is `developer.synq.io` (EU). Set `QUALITY_API_ENDPOINT` to
`api.us.synq.io` for US or `api.au.synq.io` for AU.

The program is idempotent: if you run it twice, you get the same result. So you
can run it on a schedule. At the end it reads the lineage back. Keep that step
when you adapt the example. Each mistake in this area gives a write that looks
valid and an empty graph, and reading the lineage back is the only way to see
the difference. Lineage is computed asynchronously, so the read polls.

## The files

| File | What is in it |
|---|---|
| `metabase.go` | The BI tool's own metadata and the sample content. Read this file first |
| `main.go` | Authentication, and the order of the steps. Three steps need an earlier step to be complete |
| `sync.go` | The mapping: one function per topic, each with the reason for its choice |
| `verify.go` | Reading the graph back |

The metadata is declared in the code, so the example runs with only Coalesce
Quality credentials. A real integration gets the metadata from the tool. The
[`omni_types`](../omni_types) example shows that part.

## Adapting it

Decide these four things for your own tool. They are in order of how much it
costs to get them wrong:

1. Entity IDs. Build them from the stable ID in the source system, never from a
   title. Everything is attached to the ID. If you change an ID, you create a
   second entity, and the first one is not renamed.
2. Type traits. `is_model`, `is_bi_like` and `is_test_type` control how your
   entities rank, sort and behave in the whole product. Without them, the
   catalog lists your entities but does not treat them as models, BI content or
   checks.
3. One entity group that holds all the entities you own. The group makes
   deletion work: when content is deleted in the BI tool, you stop sending it,
   and the server deletes it. Without a group, you must remember on your side
   what you created in the last run, and that record will drift from the real
   state. Send the full set on every run. A partial send deletes everything you
   leave out.
4. Check categories: leave them unset. Your workspace's own categorisation rules
   set `category` and `governance_category`, and a value sent by a producer
   overrides those rules. If you derive one from the tool's alert kind, every
   rule that the workspace wrote stops applying, and nothing tells you. Send
   `package` and `kind`, which is what the tool reports, and let the rules
   decide.

## Support

For questions about the API, or about how to model your own tool, contact your
Technical Account Manager. See [Support](https://docs.synq.io/support/support).
