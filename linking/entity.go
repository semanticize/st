// Must remember to try the following with Go >= 1.3; on Go 1.2, it only
// makes things slower.
// go:generate ffjson -nodecoder=true $GOFILE

package linking

// Represents a mention of an entity.
type Entity struct {
	// Title of target Wikipedia article.
	Target string `json:"target"`

	// Raw n-gram count estimate.
	NGramCount float64 `json:"ngramcount"`

	// Total number of links to Target in Wikipedia.
	LinkCount float64 `json:"linkcount"`

	Commonness float64 `json:"commonness"`
	Senseprob  float64 `json:"senseprob"`

	// Offset of anchor in input string.
	Offset int `json:"offset"`

	// Length of anchor in input string.
	Length int `json:"length"`
}
