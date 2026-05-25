package update

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"where-is-my-comic-service/search-services/api/core"
	updatepb "where-is-my-comic-service/search-services/proto/update"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
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
		client: updatepb.NewUpdateClient(conn),
		log:    log,
	}, nil
}

func (c Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	return err
}

func (c Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	serviceStatus, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		return "", err
	}
	status := FromProtoStatus(serviceStatus.Status)
	return status, nil
}

func (c Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	stats, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		return core.UpdateStats{}, err
	}
	return core.UpdateStats{
		WordsTotal:    int(stats.WordsTotal),
		WordsUnique:   int(stats.WordsUnique),
		ComicsFetched: int(stats.ComicsFetched),
		ComicsTotal:   int(stats.ComicsTotal),
	}, nil
}

func (c Client) Update(ctx context.Context) (core.UpdateResult, error) {
	updateReply, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		return core.UpdateResult{}, mapGRPCError(err)
	}
	return core.UpdateResult{
		ComicsInserted: int(updateReply.ComicsInserted),
	}, nil
}

func (c Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	return nil
}

func (c Client) Close() error {
	return c.conn.Close()
}
