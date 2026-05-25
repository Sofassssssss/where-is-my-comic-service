package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"where-is-my-comic-service/search-services/api/core"
)

func handleMappedError(w http.ResponseWriter, err error) {
	httpStatus, message := HttpError(err)
	http.Error(w, message, httpStatus)
}

func NewMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		promhttp.Handler().ServeHTTP(w, r)
	}
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		statuses := make(map[string]string)
		for name, service := range pingers {
			if err := service.Ping(r.Context()); err != nil {
				log.Error("Ping failed", "service", service, "error", err)
				statuses[name] = "unavailable"
				continue
			}
			statuses[name] = "ok"
		}
		resp := PingResponse{Replies: statuses}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Failed to encode ping response", "error", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
		}
	}
}

type Authenticator interface {
	Login(user, password string) (string, error)
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Error("failed to close body", "err", err)
			}
		}()
		var req LoginRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Error("Error while decoding request to json", "error", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		tokenString, err := auth.Login(req.Name, req.Password)
		if err != nil {
			log.Error("Login failed", "error", err)
			handleMappedError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		_, _ = w.Write([]byte(tokenString))
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateReply, err := updater.Update(r.Context())
		if err != nil {
			log.Error("Error", "is", err)
			handleMappedError(w, err)
			return
		}
		resp := UpdateResponse{
			ComicsInserted: updateReply.ComicsInserted,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error while encoding response to json", "error", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
			return
		}
	}
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := updater.Stats(r.Context())
		if err != nil {
			handleMappedError(w, err)
			return
		}
		resp := StatsResponse{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error while encoding response to json", "error", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
			return
		}
	}
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := updater.Status(r.Context())
		if err != nil {
			handleMappedError(w, err)
			return
		}
		resp := StatusResponse{
			Status: string(status),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error while encoding response to json", "error", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
			return
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Drop(r.Context())
		if err != nil {
			handleMappedError(w, err)
			return
		}
	}
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if len(phrase) == 0 {
			http.Error(w, "phrase cannot be empty", http.StatusBadRequest)
			return
		}
		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "invalid limit: must be integer", http.StatusBadRequest)
				return
			}
			if parsedLimit < 1 {
				http.Error(w, "invalid limit: must be greater than 0", http.StatusBadRequest)
				return
			}
			limit = parsedLimit
		}
		searchReply, err := searcher.Search(r.Context(), core.SearchRequest{
			Phrase: phrase,
			Limit:  limit,
		})
		if err != nil {
			handleMappedError(w, err)
			return
		}

		resp := SearchResponse{
			Comics: MapToSliceComicsDTO(searchReply),
			Total:  len(searchReply),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error while encoding response to json", "error", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
			return
		}
	}
}

func NewISearchHandler(log *slog.Logger, iSearcher core.ISearcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if len(phrase) == 0 {
			http.Error(w, "phrase cannot be empty", http.StatusBadRequest)
			return
		}
		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			parsedLimit, err := strconv.Atoi(limitStr)
			if err != nil {
				http.Error(w, "invalid limit: must be integer", http.StatusBadRequest)
				return
			}
			if parsedLimit < 1 {
				http.Error(w, "invalid limit: must be greater than 0", http.StatusBadRequest)
				return
			}
			limit = parsedLimit
		}
		iSearchReply, err := iSearcher.ISearch(r.Context(), core.ISearchRequest{
			Phrase: phrase,
			Limit:  limit,
		})
		if err != nil {
			handleMappedError(w, err)
			return
		}

		resp := ISearchResponse{
			Comics: MapToSliceComicsDTO(iSearchReply),
			Total:  len(iSearchReply),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("Error while encoding response to json", "error", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
			return
		}
	}
}
