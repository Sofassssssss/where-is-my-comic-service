package words

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"where-is-my-comic-service/search-services/api/core"
)

func MapGRPCError(err error) error {
	st, ok := status.FromError(err)
	if !ok {
		return core.ErrInternal
	}
	switch st.Code() {
	case codes.ResourceExhausted:
		return core.ErrBadRequest
	case codes.Internal:
		return core.ErrInternal
	default:
		return core.ErrInternal
	}
}
