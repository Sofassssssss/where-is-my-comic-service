package core

type ComicsKeyWords struct {
	ID       int
	KeyWords []string
}

type SearchRequest struct {
	Phrase string
	Limit  int
}

type Comics struct {
	ID  int64
	URL string
}
