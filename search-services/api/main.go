package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"where-is-my-comic-service/search-services/api/adapters/aaa"
	"where-is-my-comic-service/search-services/api/adapters/isearch"
	"where-is-my-comic-service/search-services/api/adapters/rest"
	"where-is-my-comic-service/search-services/api/adapters/rest/middleware"
	"where-is-my-comic-service/search-services/api/adapters/search"
	"where-is-my-comic-service/search-services/api/adapters/update"
	"where-is-my-comic-service/search-services/api/adapters/words"
	"where-is-my-comic-service/search-services/api/closers"
	"where-is-my-comic-service/search-services/api/config"
	"where-is-my-comic-service/search-services/api/core"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		log.Error("Cannot init words adapter", "error", err)
		os.Exit(1)
	}

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		log.Error("cannot init update adapter", "error", err)
		os.Exit(1)
	}

	searchClient, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		log.Error("Cannot init search adapter", "error", err)
		os.Exit(1)
	}

	iSearchClient, err := isearch.NewClient(cfg.ISearchAddress, log)
	if err != nil {
		log.Error("Cannot init search adapter", "error", err)
		os.Exit(1)
	}

	authClient, err := aaa.New(cfg.TokenTTL, log)
	if err != nil {
		log.Error("Cannot init auth adapter", "error", err)
		os.Exit(1)
	}

	fs := http.FileServer(http.Dir("./frontend"))

	middleware.Init()

	defer closers.CloseOrLog(wordsClient, log)
	defer closers.CloseOrLog(updateClient, log)
	defer closers.CloseOrLog(searchClient, log)
	defer closers.CloseOrLog(iSearchClient, log)

	mux := http.NewServeMux()

	services := map[string]core.Pinger{"words": wordsClient, "update": updateClient, "search": searchClient, "isearch": iSearchClient}

	mux.Handle("GET /api/ping", rest.NewPingHandler(log, services))
	mux.Handle("POST /api/db/update", middleware.Auth(rest.NewUpdateHandler(log, updateClient), authClient))
	mux.Handle("GET /api/db/stats", rest.NewUpdateStatsHandler(log, updateClient))
	mux.Handle("GET /api/db/status", rest.NewUpdateStatusHandler(log, updateClient))
	mux.Handle("DELETE /api/db", middleware.Auth(rest.NewDropHandler(log, updateClient), authClient))
	mux.Handle("GET /api/search", middleware.Concurrency(rest.NewSearchHandler(log, searchClient), cfg.SearchConcurrency))
	mux.Handle("GET /api/isearch", middleware.Rate(rest.NewISearchHandler(log, iSearchClient), cfg.SearchRate))
	mux.Handle("POST /api/login", rest.NewLoginHandler(log, authClient))
	mux.Handle("GET /metrics", rest.NewMetricsHandler())
	mux.Handle("/", fs)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	handler := middleware.WithMetrics(mux)
	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     handler,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
			return
		}
	}
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
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level, AddSource: true})
	return slog.New(handler)
}
