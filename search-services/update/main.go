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
	updatepb "where-is-my-comic-service/search-services/proto/update"
	"where-is-my-comic-service/search-services/update/adapters/db"
	updategrpc "where-is-my-comic-service/search-services/update/adapters/grpc"
	"where-is-my-comic-service/search-services/update/adapters/nats"
	"where-is-my-comic-service/search-services/update/adapters/words"
	"where-is-my-comic-service/search-services/update/adapters/xkcd"
	"where-is-my-comic-service/search-services/update/closers"
	"where-is-my-comic-service/search-services/update/config"
	"where-is-my-comic-service/search-services/update/core"
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

	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	// xkcd adapter
	xkcd, err := xkcd.NewClient(cfg.XKCD.URL, cfg.XKCD.Timeout, log)
	if err != nil {
		return fmt.Errorf("failed create XKCD client: %v", err)
	}

	// words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}

	publisher, err := nats.NewPublisher(cfg.NatsAddress)
	if err != nil {
		log.Error("failed to connect to nats", "err", err)
		os.Exit(1)
	}

	// service
	updater, err := core.NewService(log, storage, xkcd, words, publisher, cfg.XKCD.Concurrency)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	defer closers.CloseOrLog(words, log)
	defer publisher.Close()

	s := grpc.NewServer()
	updatepb.RegisterUpdateServer(s, updategrpc.NewServer(updater))
	reflection.Register(s)

	// context for Ctrl-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

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
