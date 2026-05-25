package initiator

import (
	"context"
	"log/slog"
	"time"

	"where-is-my-comic-service/search-services/isearch/core"
)

type Initiator struct {
	builder core.IndexBuilder
	ttl     time.Duration
	log     *slog.Logger
}

func New(builder core.IndexBuilder, ttl time.Duration, log *slog.Logger) *Initiator {
	return &Initiator{
		builder: builder,
		ttl:     ttl,
		log:     log,
	}
}

func (i *Initiator) InitInitial(ctx context.Context) error {
	i.log.Info("Starting initial index build...")
	if err := i.builder.RebuildIndex(ctx); err != nil {
		i.log.Error("Failed to build initial index", "err", err)
		return err
	}
	i.log.Info("Initial index built successfully")
	return nil
}

func (i *Initiator) Start(ctx context.Context) {
	ticker := time.NewTicker(i.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			i.log.Info("Stopping initiator...")
			return
		case <-ticker.C:
			i.log.Info("Rebuilding index by timer...")
			if err := i.builder.RebuildIndex(ctx); err != nil {
				i.log.Error("Failed to rebuild index", "err", err)
			}
		}
	}
}
