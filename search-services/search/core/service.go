package core

import (
	"context"
	"log/slog"
	"strings"

	"github.com/iwilltry42/bm25-go/bm25"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
}

func NewService(log *slog.Logger, db DB, words Words) (*Service, error) {
	return &Service{
		log:   log,
		db:    db,
		words: words,
	}, nil
}

func (s *Service) Search(ctx context.Context, req SearchRequest) ([]Comics, error) {
	normalizedPhrase, err := s.words.Norm(ctx, req.Phrase)
	if err != nil {
		return nil, err
	}

	comicsData, err := s.db.GetComicsData(ctx)
	if err != nil {
		return nil, err
	}
	corpus := make([]string, 0, len(comicsData))
	comicsID := make([]int, 0, len(comicsData))

	for _, c := range comicsData {
		corpus = append(corpus, strings.Join(c.KeyWords, " "))
		comicsID = append(comicsID, c.ID)
	}

	if len(corpus) == 0 {
		return []Comics{}, nil
	}

	tokenizer := func(s string) []string {
		return strings.Split(s, " ")
	}

	bm, err := bm25.NewBM25Okapi(corpus, tokenizer, 1.5, 0.75, nil)
	if err != nil {
		return nil, err
	}
	scores, err := bm.GetScores(normalizedPhrase)
	if err != nil {
		return nil, err
	}
	topIndices, err := bm25.TopNIndices(scores, req.Limit)
	if err != nil {
		return nil, err
	}
	var result []Comics

	for _, index := range topIndices {
		if scores[index] == 0 {
			continue
		}
		comicsImageURL, err := s.db.GetImageURL(ctx, comicsID[index])
		if err != nil {
			return nil, err
		}
		result = append(result, Comics{
			ID:  int64(comicsID[index]),
			URL: comicsImageURL,
		})
	}
	return result, nil
}
