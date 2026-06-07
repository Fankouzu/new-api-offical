package webseo

type Meta struct {
	Title        string
	Description  string
	CanonicalURL string
	Robots       string
	OGType       string
	JSONLD       []map[string]any
}

type ModelSEOItem struct {
	ID           string
	Name         string
	Description  string
	Family       string
	InputPrice   float64
	OutputPrice  float64
	Capabilities []string
}
