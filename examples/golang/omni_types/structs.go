package main

type DocumentsResponse struct {
	Records []*Document `json:"records"`
}

type DocumentExportResponse struct {
	Dashboard *Dashboard `json:"dashboard"`
	Document  *Document  `json:"document"`
}

type Document struct {
	Identifier string      `json:"identifier"`
	Connection *Connection `json:"connection"`
}

type Dashboard struct {
	Name                        string                       `json:"name"`
	QueryPresentationCollection *QueryPresentationCollection `json:"queryPresentationCollection"`
}

type QueryPresentationCollection struct {
	QueryPresentationCollectionMemberships []*QueryPresentationCollectionMembership `json:"queryPresentationCollectionMemberships"`
}

type QueryPresentationCollectionMembership struct {
	QueryPresentation *QueryPresentation `json:"queryPresentation"`
}

type QueryPresentation struct {
	Query *Query `json:"query"`
}

type Query struct {
	QueryJson *QueryJson `json:"queryJson"`
}

type QueryJson struct {
	Table  string   `json:"table"`
	Fields []string `json:"fields"`
}

type Connection struct {
	Database string `json:"database"`
	Dialect  string `json:"dialect"`
	Id       string `json:"id"`
	Name     string `json:"name"`
}
