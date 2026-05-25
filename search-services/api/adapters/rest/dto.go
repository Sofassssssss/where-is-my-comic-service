package rest

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

type WordsResponse struct {
	Words []string `json:"words"`
	Total int      `json:"total"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type StatsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type UpdateResponse struct {
	ComicsInserted int `json:"comics_inserted"`
}

type ComicsDTO struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type SearchResponse struct {
	Comics []ComicsDTO `json:"comics"`
	Total  int         `json:"total"`
}

type ISearchResponse struct {
	Comics []ComicsDTO `json:"comics"`
	Total  int         `json:"total"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
