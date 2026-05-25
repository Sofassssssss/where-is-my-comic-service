package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"where-is-my-comic-service/search-services/isearch/adapters/db"
	isearchgrpc "where-is-my-comic-service/search-services/isearch/adapters/grpc"
	"where-is-my-comic-service/search-services/isearch/adapters/initiator"
	"where-is-my-comic-service/search-services/isearch/adapters/nats"
	"where-is-my-comic-service/search-services/isearch/adapters/words"
	"where-is-my-comic-service/search-services/isearch/closers"
	"where-is-my-comic-service/search-services/isearch/config"
	"where-is-my-comic-service/search-services/isearch/core"
	isearchpb "where-is-my-comic-service/search-services/proto/isearch"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}

	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}

	defer closers.CloseOrLog(words, log)

	// service
	isearcher, err := core.NewService(log, storage, words)
	if err != nil {
		return fmt.Errorf("failed create ISearch service: %v", err)
	}

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	isearchpb.RegisterISearchServer(s, isearchgrpc.NewServer(isearcher))
	reflection.Register(s)

	// context for Ctrl-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	indexInitiator := initiator.New(isearcher, cfg.IndexTTL, log)
	if err := indexInitiator.InitInitial(ctx); err != nil {
		log.Error("Failed to initialize index, but starting server anyway", "err", err)

	}

	go indexInitiator.Start(ctx)

	subscriber, err := nats.NewSubscriber(cfg.NatsAddress)
	if err != nil {
		log.Error("failed init subscriber", "err", err)
	}

	defer subscriber.Close()

	if err != nil {
		log.Error("failed to connect to nats", "err", err)
	}

	sub, err := subscriber.Subscribe(func() {
		go func() {
			log.Info("received db updated event")
			ctx := context.Background()
			err := isearcher.RebuildIndex(ctx)
			if err != nil {
				log.Error("failed to rebuild index", "err", err)
			}
		}()
	})

	if err != nil {
		log.Error("failed to subscribe", "err", err)
	}

	defer func() {
		err := sub.Unsubscribe()
		if err != nil {
			log.Error("failed to unsubscribe", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
