package update

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"where-is-my-comic-service/search-services/api/core"
	"where-is-my-comic-service/search-services/proto/update"
)

func FromProtoStatus(s update.Status) core.UpdateStatus {
	switch s {
	case update.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning

	case update.Status_STATUS_IDLE:
		return core.StatusUpdateIdle

	default:
		return core.StatusUpdateUnknown
	}
}

func mapGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return core.ErrInternal
	}

	switch st.Code() {
	case codes.Unavailable:
		return core.ErrUpdateRunning

	case codes.InvalidArgument:
		return core.ErrBadRequest

	default:
		return core.ErrInternal
	}
}
