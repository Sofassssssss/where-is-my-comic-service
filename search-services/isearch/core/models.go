package core

type ComicsKeyWords struct {
	ID       int
	KeyWords []string
}

type ISearchRequest struct {
	Phrase string
	Limit  int
}

type Comics struct {
	ID  int64
	URL string
}

type DocMeta struct {
	DocID int
	TF    int // Term Frequency
}

type SearchIndex struct {
	InvertedIndex map[string][]DocMeta
	IDF           map[string]float64 // Inverse Document Frequency
	DocLengths    []int
	TotalDocs     int
	AvgDocLength  float64
}
