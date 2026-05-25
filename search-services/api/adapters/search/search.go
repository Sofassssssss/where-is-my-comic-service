package search

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"where-is-my-comic-service/search-services/api/core"

	"log/slog"
	"time"

	"google.golang.org/grpc/backoff"
	searchpb "where-is-my-comic-service/search-services/proto/search"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
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
		client: searchpb.NewSearchClient(conn),
		conn:   conn,
		log:    log,
	}, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c Client) Search(ctx context.Context, req core.SearchRequest) ([]core.Comics, error) {
	searchReply, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Phrase: req.Phrase,
		Limit:  int32(req.Limit),
	})
	if err != nil {
		return nil, MapGRPCError(err)
	}

	result := make([]core.Comics, 0, len(searchReply.Comics))

	for _, comics := range searchReply.Comics {
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
