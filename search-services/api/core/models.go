package core

type UpdateStatus string

const (
	StatusUpdateUnknown UpdateStatus = "unknown"
	StatusUpdateIdle    UpdateStatus = "idle"
	StatusUpdateRunning UpdateStatus = "running"
)

type UpdateStats struct {
	WordsTotal    int
	WordsUnique   int
	ComicsFetched int
	ComicsTotal   int
}

type UpdateResult struct {
	ComicsInserted int
}

type Comics struct {
	ID  int
	URL string
}

type SearchRequest struct {
	Phrase string
	Limit  int
}

type ISearchRequest struct {
	Phrase string
	Limit  int
}
