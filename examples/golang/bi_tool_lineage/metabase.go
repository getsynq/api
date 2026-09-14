package main

// -----------------------------------------------------------------------------
// The BI tool's own metadata — the input half of the example.
// -----------------------------------------------------------------------------
//
// Metabase stands in for "a BI tool Coalesce Quality has no integration for".
// It is a real product, and its object model is the one most BI tools have under
// different names:
//
//	Metabase     Looker        Sigma           what it is
//	---------    -----------   -------------   --------------------------------
//	model        view          data model      a saved query others build on
//	question     look          workbook query  a query, often built on a model
//	dashboard    dashboard     workbook        a composition of question results
//	subscription schedule      scheduled send  a delivery of a dashboard
//	alert        alert         alert           a condition watched on a question
//
// Rename the structs and the mapping in sync.go is unchanged, which is the point
// of reading this file first: modelling a new tool is a mapping exercise, not an
// API exercise.
//
// A real integration fetches all of this from the tool — for Metabase that is
// GET /api/card, GET /api/dashboard/:id and GET /api/alert. It is declared inline
// here so the example runs with nothing but Coalesce Quality credentials; the
// `omni_types` example shows the fetch half against a live BI tool.

// column is one field the tool exposes on a model, question or dashboard.
type column struct {
	name       string
	nativeType string
	desc       string
}

// model is a saved, reusable query other content is built on.
type model struct {
	id          int
	name        string
	description string
	collection  string

	// sql is the query as the tool runs it against the warehouse. Table names in
	// it are real warehouse objects, so Coalesce Quality resolves them by address
	// and no binding is needed.
	sql string

	// sqlName is the name a DOWNSTREAM query writes when it reads this model.
	// Metabase itself compiles a nested question into a subquery and never emits
	// such a name, so the integration picks one — and picks it from the model's
	// stable id rather than its title, so renaming the model in Metabase does not
	// silently repoint every binding. A readable name is used here to keep the
	// example legible.
	sqlName string

	columns []column

	// definition is the tool's own object, carried across verbatim so a reviewer
	// can see what changed between two syncs without leaving Coalesce Quality.
	definition string
}

// question is a query, optionally built on one or more models.
type question struct {
	id          int
	name        string
	description string
	collection  string

	sql string

	// sourceModels maps a name this question's SQL writes to the model behind it.
	// Every name here needs a binding, because a model is not an object the
	// warehouse can address; the names NOT listed are ordinary warehouse tables
	// and resolve on their own.
	sourceModels map[string]int

	columns []column
}

// dashboard is a composition of question results. It runs no SQL of its own: a
// field on a card is the question's column, carried through unchanged or
// relabelled, which is why its lineage is declared rather than derived.
type dashboard struct {
	id          int
	name        string
	description string
	collection  string

	cards []card
}

// card is one field of one dashboard tile, and where its value comes from.
// Exactly one of fromQuestion / fromWarehouseTable is set.
type card struct {
	// field is the dashboard's own column name.
	field string

	// fromQuestion is the question id the value comes from, and fromColumn the
	// column on it.
	fromQuestion int

	// fromWarehouseTable is set instead when a tile reads the warehouse directly
	// — a filter widget populated from a column, say — rather than through a
	// question.
	fromWarehouseTable string

	fromColumn string
}

// subscription is a scheduled delivery of a dashboard. It has no columns of its
// own, so it is joined to the dashboard by a plain relationship.
type subscription struct {
	id          int
	name        string
	description string
	dashboard   int
	schedule    string
}

// alert is a condition watched on a question. It is a check: something that
// passes or fails about another entity, rather than a stage data flows through.
type alert struct {
	id          int
	name        string
	description string

	// question is the entity this alert checks.
	question int

	// kind is the tool's own name for what the alert does. It is reported as-is
	// and deliberately not mapped to a check category — see sync.go.
	kind string
}

// -----------------------------------------------------------------------------
// The sample content.
// -----------------------------------------------------------------------------
//
// Two models over the warehouse, two questions over the models, one dashboard
// over the questions, one subscription and one alert. Small enough to follow in
// the lineage graph, wide enough that every mechanism appears once.

var models = []model{
	{
		id:          31,
		name:        "Order items enriched",
		description: "Order lines with product attributes and net revenue applied. The commercial team's starting point.",
		collection:  "Commercial",
		sqlName:     "order_items_enriched",
		sql: `SELECT
  oi.order_id                                     AS order_id,
  oi.item_id                                      AS item_id,
  oi.product_id                                   AS product_id,
  p.name                                          AS product_name,
  p.category                                      AS product_category,
  oi.quantity                                     AS quantity,
  oi.unit_price                                   AS unit_price,
  oi.quantity * oi.unit_price * (1 - oi.discount) AS net_revenue
FROM order_items AS oi
JOIN products AS p ON p.id = oi.product_id`,
		columns: []column{
			{"order_id", "INTEGER", "Order the line belongs to"},
			{"item_id", "INTEGER", "Line identifier within the order"},
			{"product_id", "INTEGER", "Product sold on this line"},
			{"product_name", "STRING", "Product name at query time"},
			{"product_category", "STRING", "Category the product belongs to"},
			{"quantity", "INTEGER", "Units sold on this line"},
			{"unit_price", "NUMERIC", "List price per unit"},
			{"net_revenue", "NUMERIC", "quantity * unit_price after the line discount"},
		},
		definition: `{"id":31,"type":"model","name":"Order items enriched","collection":"Commercial","database_id":2}`,
	},
	{
		id:          32,
		name:        "Customer regions",
		description: "One row per customer with the region used for territory reporting.",
		collection:  "Commercial",
		sqlName:     "customer_regions_v",
		sql: `SELECT
  c.id        AS customer_id,
  c.full_name AS customer_name,
  c.region    AS region
FROM customers AS c`,
		columns: []column{
			{"customer_id", "INTEGER", "Customer surrogate key"},
			{"customer_name", "STRING", "Customer display name"},
			{"region", "STRING", "Sales region"},
		},
		definition: `{"id":32,"type":"model","name":"Customer regions","collection":"Commercial","database_id":2}`,
	},
}

var questions = []question{
	{
		id:          87,
		name:        "Revenue by product category",
		description: "Net revenue and units, grouped by product category.",
		collection:  "Commercial",
		// Reads one model and nothing else, so every table name in it is bound.
		sql: `SELECT
  m.product_category AS category,
  SUM(m.net_revenue) AS revenue,
  SUM(m.quantity)    AS units
FROM order_items_enriched AS m
GROUP BY m.product_category`,
		sourceModels: map[string]int{"order_items_enriched": 31},
		columns: []column{
			{"category", "STRING", "Product category"},
			{"revenue", "NUMERIC", "Net revenue for the category"},
			{"units", "INTEGER", "Units sold in the category"},
		},
	},
	{
		id:          88,
		name:        "Customers by region",
		description: "Customer counts per region, with the contactable share.",
		collection:  "Commercial",
		// Mixes both resolution paths in one definition: `customer_regions_v` is a
		// model and is bound, `customers` is a warehouse table and is not.
		sql: `SELECT
  r.region                      AS region,
  COUNT(DISTINCT r.customer_id) AS customers,
  COUNT(DISTINCT c.email)       AS contactable_customers
FROM customer_regions_v AS r
JOIN customers AS c ON c.id = r.customer_id
GROUP BY r.region`,
		sourceModels: map[string]int{"customer_regions_v": 32},
		columns: []column{
			{"region", "STRING", "Sales region"},
			{"customers", "INTEGER", "Customers in the region"},
			{"contactable_customers", "INTEGER", "Customers with an email address on file"},
		},
	},
}

var dashboards = []dashboard{
	{
		id:          9,
		name:        "Commercial overview",
		description: "The weekly commercial read: revenue by category, customers by region.",
		collection:  "Commercial",
		cards: []card{
			{field: "revenue_by_category.category", fromQuestion: 87, fromColumn: "category"},
			{field: "revenue_by_category.revenue", fromQuestion: 87, fromColumn: "revenue"},
			{field: "revenue_by_category.units", fromQuestion: 87, fromColumn: "units"},
			{field: "customers_by_region.region", fromQuestion: 88, fromColumn: "region"},
			{field: "customers_by_region.customers", fromQuestion: 88, fromColumn: "customers"},
			// A filter widget reading the warehouse directly, without a question in
			// between. A declared edge reaches a native entity exactly as it reaches
			// a custom one.
			{field: "filter.product_category", fromWarehouseTable: "products", fromColumn: "category"},
		},
	},
}

var subscriptions = []subscription{
	{
		id:          4,
		name:        "Weekly commercial digest",
		description: "Sends the Commercial overview to the leadership list every Monday at 07:00.",
		dashboard:   9,
		schedule:    "weekly, Monday 07:00 UTC",
	},
}

var alerts = []alert{
	{
		id:          12,
		name:        "Revenue below weekly goal",
		description: "Fires when total net revenue for the week falls under the commercial goal.",
		question:    87,
		kind:        "goal_alert",
	},
}
