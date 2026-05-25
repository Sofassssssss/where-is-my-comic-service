package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net"

	wordspb "where-is-my-comic-service/search-services/proto/words"
	normalize "where-is-my-comic-service/search-services/words/words"

	"github.com/alecthomas/units"
	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

var language = "english"
var maxPhraseLen = int(4 * units.KiB)

type ServerConfig struct {
	Address string `yaml:"address" env:"WORDS_ADDRESS" env-default:":8080"`
}
type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *server) Norm(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	var result []string
	phrase := in.GetPhrase()
	if len(phrase) > maxPhraseLen {
		phrase = phrase[:maxPhraseLen]
	}
	result, err := normalize.Norm(phrase, language)
	if err != nil {
		return nil, err
	}

	return &wordspb.WordsReply{
		Words: result,
	}, nil
}

func (s *server) NormLeaveDuplicates(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	var result []string
	phrase := in.GetPhrase()
	phraseForLog := in.Phrase
	if len(phraseForLog) > 50 {
		phraseForLog = phraseForLog[:50]
	}

	slog.Info("NormLeaveDuplicates Request ", "request", phraseForLog)
	if len(phrase) > maxPhraseLen {
		phrase = phrase[:maxPhraseLen]
	}
	result, err := normalize.NormLeaveDuplicates(phrase, language)
	if err != nil {
		return nil, err
	}

	return &wordspb.WordsReply{
		Words: result,
	}, nil
}

func main() {
	var cfg ServerConfig
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		slog.Error("Error while reading config" + err.Error())
		return
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	slog.Info("Server started at " + cfg.Address)
	wordspb.RegisterWordsServer(s, &server{})
	reflection.Register(s)

	if err := s.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
