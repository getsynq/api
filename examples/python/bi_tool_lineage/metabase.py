"""
The BI tool's own metadata — the input half of the example.

Metabase stands in for "a BI tool Coalesce Quality has no integration for". It is
a real product, and its object model is the one most BI tools have under
different names:

    Metabase     Looker        Sigma           what it is
    ---------    -----------   -------------   --------------------------------
    model        view          data model      a saved query others build on
    question     look          workbook query  a query, often built on a model
    dashboard    dashboard     workbook        a composition of question results
    subscription schedule      scheduled send  a delivery of a dashboard
    alert        alert         alert           a condition watched on a question

Rename the dataclasses and the mapping in sync.py is unchanged, which is the
point of reading this file first: modelling a new tool is a mapping exercise, not
an API exercise.

A real integration fetches all of this from the tool — for Metabase that is
GET /api/card, GET /api/dashboard/:id and GET /api/alert. It is declared inline
here so the example runs with nothing but Coalesce Quality credentials; the
`integrations_management` example shows the fetch half against a live system.
"""

from dataclasses import dataclass, field
from typing import Dict, List, Optional


@dataclass
class Column:
    """One field the tool exposes on a model, question or dashboard."""

    name: str
    native_type: str = ""
    desc: str = ""


@dataclass
class Model:
    """A saved, reusable query other content is built on."""

    id: int
    name: str
    description: str
    collection: str

    # The query as the tool runs it against the warehouse. Table names in it are
    # real warehouse objects, so Coalesce Quality resolves them by address and no
    # binding is needed.
    sql: str

    # The name a DOWNSTREAM query writes when it reads this model. Metabase
    # itself compiles a nested question into a subquery and never emits such a
    # name, so the integration picks one — and picks it from the model's stable
    # id rather than its title, so renaming the model in Metabase does not
    # silently repoint every binding. A readable name is used here to keep the
    # example legible.
    sql_name: str

    columns: List[Column]

    # The tool's own object, carried across verbatim so a reviewer can see what
    # changed between two syncs without leaving Coalesce Quality.
    definition: str


@dataclass
class Question:
    """A query, optionally built on one or more models."""

    id: int
    name: str
    description: str
    collection: str
    sql: str

    # Maps a name this question's SQL writes to the model behind it. Every name
    # here needs a binding, because a model is not an object the warehouse can
    # address; the names NOT listed are ordinary warehouse tables and resolve on
    # their own.
    source_models: Dict[str, int]

    columns: List[Column]


@dataclass
class Card:
    """
    One field of one dashboard tile, and where its value comes from.

    Exactly one of from_question / from_warehouse_table is set.
    """

    field: str
    from_column: str
    from_question: Optional[int] = None
    # Set instead when a tile reads the warehouse directly — a filter widget
    # populated from a column, say — rather than through a question.
    from_warehouse_table: Optional[str] = None


@dataclass
class Dashboard:
    """
    A composition of question results.

    It runs no SQL of its own: a field on a card is the question's column,
    carried through unchanged or relabelled, which is why its lineage is declared
    rather than derived.
    """

    id: int
    name: str
    description: str
    collection: str
    cards: List[Card] = field(default_factory=list)


@dataclass
class Subscription:
    """
    A scheduled delivery of a dashboard.

    It has no columns of its own, so it is joined to the dashboard by a plain
    relationship.
    """

    id: int
    name: str
    description: str
    dashboard: int
    schedule: str


@dataclass
class Alert:
    """
    A condition watched on a question.

    It is a check: something that passes or fails about another entity, rather
    than a stage data flows through.
    """

    id: int
    name: str
    description: str
    question: int
    # The tool's own name for what the alert does. Reported as-is and
    # deliberately not mapped to a check category — see sync.py.
    kind: str


# -----------------------------------------------------------------------------
# The sample content.
# -----------------------------------------------------------------------------
#
# Two models over the warehouse, two questions over the models, one dashboard
# over the questions, one subscription and one alert. Small enough to follow in
# the lineage graph, wide enough that every mechanism appears once.

MODELS = [
    Model(
        id=31,
        name="Order items enriched",
        description=(
            "Order lines with product attributes and net revenue applied. "
            "The commercial team's starting point."
        ),
        collection="Commercial",
        sql_name="order_items_enriched",
        sql="""SELECT
  oi.order_id                                     AS order_id,
  oi.item_id                                      AS item_id,
  oi.product_id                                   AS product_id,
  p.name                                          AS product_name,
  p.category                                      AS product_category,
  oi.quantity                                     AS quantity,
  oi.unit_price                                   AS unit_price,
  oi.quantity * oi.unit_price * (1 - oi.discount) AS net_revenue
FROM order_items AS oi
JOIN products AS p ON p.id = oi.product_id""",
        columns=[
            Column("order_id", "INTEGER", "Order the line belongs to"),
            Column("item_id", "INTEGER", "Line identifier within the order"),
            Column("product_id", "INTEGER", "Product sold on this line"),
            Column("product_name", "STRING", "Product name at query time"),
            Column("product_category", "STRING", "Category the product belongs to"),
            Column("quantity", "INTEGER", "Units sold on this line"),
            Column("unit_price", "NUMERIC", "List price per unit"),
            Column("net_revenue", "NUMERIC", "quantity * unit_price after the line discount"),
        ],
        definition=(
            '{"id":31,"type":"model","name":"Order items enriched",'
            '"collection":"Commercial","database_id":2}'
        ),
    ),
    Model(
        id=32,
        name="Customer regions",
        description="One row per customer with the region used for territory reporting.",
        collection="Commercial",
        sql_name="customer_regions_v",
        sql="""SELECT
  c.id        AS customer_id,
  c.full_name AS customer_name,
  c.region    AS region
FROM customers AS c""",
        columns=[
            Column("customer_id", "INTEGER", "Customer surrogate key"),
            Column("customer_name", "STRING", "Customer display name"),
            Column("region", "STRING", "Sales region"),
        ],
        definition=(
            '{"id":32,"type":"model","name":"Customer regions",'
            '"collection":"Commercial","database_id":2}'
        ),
    ),
]

QUESTIONS = [
    Question(
        id=87,
        name="Revenue by product category",
        description="Net revenue and units, grouped by product category.",
        collection="Commercial",
        # Reads one model and nothing else, so every table name in it is bound.
        sql="""SELECT
  m.product_category AS category,
  SUM(m.net_revenue) AS revenue,
  SUM(m.quantity)    AS units
FROM order_items_enriched AS m
GROUP BY m.product_category""",
        source_models={"order_items_enriched": 31},
        columns=[
            Column("category", "STRING", "Product category"),
            Column("revenue", "NUMERIC", "Net revenue for the category"),
            Column("units", "INTEGER", "Units sold in the category"),
        ],
    ),
    Question(
        id=88,
        name="Customers by region",
        description="Customer counts per region, with the contactable share.",
        collection="Commercial",
        # Mixes both resolution paths in one definition: `customer_regions_v` is
        # a model and is bound, `customers` is a warehouse table and is not.
        sql="""SELECT
  r.region                      AS region,
  COUNT(DISTINCT r.customer_id) AS customers,
  COUNT(DISTINCT c.email)       AS contactable_customers
FROM customer_regions_v AS r
JOIN customers AS c ON c.id = r.customer_id
GROUP BY r.region""",
        source_models={"customer_regions_v": 32},
        columns=[
            Column("region", "STRING", "Sales region"),
            Column("customers", "INTEGER", "Customers in the region"),
            Column("contactable_customers", "INTEGER", "Customers with an email address on file"),
        ],
    ),
]

DASHBOARDS = [
    Dashboard(
        id=9,
        name="Commercial overview",
        description="The weekly commercial read: revenue by category, customers by region.",
        collection="Commercial",
        cards=[
            Card("revenue_by_category.category", "category", from_question=87),
            Card("revenue_by_category.revenue", "revenue", from_question=87),
            Card("revenue_by_category.units", "units", from_question=87),
            Card("customers_by_region.region", "region", from_question=88),
            Card("customers_by_region.customers", "customers", from_question=88),
            # A filter widget reading the warehouse directly, without a question
            # in between. A declared edge reaches a native entity exactly as it
            # reaches a custom one.
            Card("filter.product_category", "category", from_warehouse_table="products"),
        ],
    ),
]

SUBSCRIPTIONS = [
    Subscription(
        id=4,
        name="Weekly commercial digest",
        description="Sends the Commercial overview to the leadership list every Monday at 07:00.",
        dashboard=9,
        schedule="weekly, Monday 07:00 UTC",
    ),
]

ALERTS = [
    Alert(
        id=12,
        name="Revenue below weekly goal",
        description="Fires when total net revenue for the week falls under the commercial goal.",
        question=87,
        kind="goal_alert",
    ),
]
