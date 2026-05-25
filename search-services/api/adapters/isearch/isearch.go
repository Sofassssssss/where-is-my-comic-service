package isearch

import (
	"context"

	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"where-is-my-comic-service/search-services/api/core"
	isearchpb "where-is-my-comic-service/search-services/proto/isearch"
)

type Client struct {
	log    *slog.Logger
	client isearchpb.ISearchClient
	conn   *grpc.ClientConn
}

var connectParams = grpc.ConnectParams{
	Backoff: backoff.Config{
		BaseDelay:  1 * time.Second,
		Multiplier: 1.6,
		MaxDelay:   10 * time.Second,
	},
	MinConnectTimeout: 2 * time.Second,
}

var retryPolicy = `
{
	"methodConfig": [{
			"name": [{"service": "search.Words", "method": ""}],
			"retryPolicy": {
				"maxAttempts": 4,
				"initialBackoff": "1s",
				"maxBackoff": "5s",
				"backoffMultiplier": 2.0,
				"retryableStatusCodes": ["UNAVAILABLE", "INTERNAL", "UNKNOWN"]
			}
		}]
}`

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(connectParams),
		grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: isearchpb.NewISearchClient(conn),
		conn:   conn,
		log:    log,
	}, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c Client) ISearch(ctx context.Context, req core.ISearchRequest) ([]core.Comics, error) {
	iSearchReply, err := c.client.ISearch(ctx, &isearchpb.ISearchRequest{
		Phrase: req.Phrase,
		Limit:  int32(req.Limit),
	})
	if err != nil {
		return nil, MapGRPCError(err)
	}

	result := make([]core.Comics, 0, len(iSearchReply.Comics))

	for _, comics := range iSearchReply.Comics {
		result = append(result, core.Comics{
			ID:  int(comics.Id),
			URL: comics.Url,
		})
	}
	return result, nil
}

func (c Client) Close() error {
	return c.conn.Close()
}
