package grpc

import (
	"where-is-my-comic-service/search-services/proto/update"
	"where-is-my-comic-service/search-services/update/core"
)

func ToProtoStatus(s core.ServiceStatus) update.Status {
	switch s {
	case core.StatusRunning:
		return update.Status_STATUS_RUNNING

	case core.StatusIdle:
		return update.Status_STATUS_IDLE

	default:
		return update.Status_STATUS_UNSPECIFIED
	}
}
