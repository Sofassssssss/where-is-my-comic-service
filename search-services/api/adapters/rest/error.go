package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"where-is-my-comic-service/search-services/api/core"
)

func HttpError(err error) (int, string) {
	switch {
	case errors.Is(err, core.ErrBadRequest):
		return http.StatusBadRequest, err.Error() + " check query parameters"
	case errors.Is(err, core.ErrInternal):
		return http.StatusInternalServerError, "internal server error"
	case errors.Is(err, core.ErrUpdateRunning):
		return http.StatusAccepted, "update already running"
	case errors.Is(err, core.ErrUnavailable):
		return http.StatusServiceUnavailable, "error on server side"
	case errors.Is(err, core.ErrIndexNotReady):
		return http.StatusServiceUnavailable, "index is not ready"
	case errors.Is(err, core.ErrUnauthorized):
		return http.StatusUnauthorized, "error unauthorized"
	default:
		slog.Debug("Unknown error", "err", err)
		return http.StatusInternalServerError, "unknown error"
	}
}
