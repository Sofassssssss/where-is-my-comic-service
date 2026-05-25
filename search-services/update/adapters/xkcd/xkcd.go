package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"where-is-my-comic-service/search-services/update/core"
)

var apiBasePath = "/info.0.json"

type APIGetResponse struct {
	ID          int    `json:"num"`
	URL         string `json:"img"`
	Title       string `json:"title"`
	SafeTitle   string `json:"safe_title"`
	Description string `json:"transcript"`
	Alt         string `json:"alt"`
}

type APILastIDResponse struct {
	LastID int `json:"num"`
}

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	resp, err := http.Get(c.url + "/" + strconv.Itoa(id) + apiBasePath)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("error while getting response from url")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("Error while closing body", "err", err)
		}
	}()
	if resp.StatusCode == http.StatusNotFound {
		return core.XKCDInfo{
			ID:          -1,
			URL:         "",
			Title:       "",
			SafeTitle:   "",
			Description: "",
			Alt:         "",
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("error while getting body from url")
	}
	var apiResponse APIGetResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return core.XKCDInfo{}, fmt.Errorf("error while unmarshall api response")
	}
	return core.XKCDInfo{
		ID:          apiResponse.ID,
		URL:         apiResponse.URL,
		Title:       apiResponse.Title,
		SafeTitle:   apiResponse.SafeTitle,
		Description: apiResponse.Description,
		Alt:         apiResponse.Alt,
	}, nil
}

func (c Client) LastID(ctx context.Context) (int, error) {
	resp, err := http.Get(c.url + apiBasePath)
	if err != nil {
		return 0, fmt.Errorf("error while getting response from url")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Error("Error while closing body", "err", err)
		}
	}()
	var apiResponse APILastIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return 0, fmt.Errorf("error while decoding response body")
	}
	return apiResponse.LastID, nil
}
