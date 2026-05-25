package core

import (
	"context"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/iwilltry42/bm25-go/bm25"
)

const k1 = 1.5
const b = 0.75

type Service struct {
	log        *slog.Logger
	db         DB
	words      Words
	index      atomic.Pointer[SearchIndex]
	rebuilding atomic.Bool
}

type searchResult struct {
	ID    int
	Score float64
}

func NewService(log *slog.Logger, db DB, words Words) (*Service, error) {
	return &Service{
		log:   log,
		db:    db,
		words: words,
	}, nil
}

func removeDuplicates(keyWords []string) []string {
	slices.Sort(keyWords)
	keyWords = slices.Compact(keyWords)
	return keyWords
}

func tokenizer(s string) []string {
	return strings.Split(s, " ")
}

func buildSearchIndex(comicsData []ComicsKeyWords) (*SearchIndex, error) {
	searchIndex := &SearchIndex{
		InvertedIndex: make(map[string][]DocMeta),
		IDF:           make(map[string]float64),
	}
	corpus := make([]string, 0, len(comicsData))

	for _, c := range comicsData {
		corpus = append(corpus, strings.Join(c.KeyWords, " "))
	}

	if len(corpus) == 0 {
		return searchIndex, nil
	}
	bm, err := bm25.NewBM25Okapi(corpus, tokenizer, k1, b, nil)
	if err != nil {
		return nil, err
	}
	for _, data := range comicsData {
		keyWordsJoined := strings.Join(data.KeyWords, " ")
		uniqueKeyWords := removeDuplicates(data.KeyWords)
		for _, keyWord := range uniqueKeyWords {
			termFreq, err := bm25.CountTermFreq(keyWord, keyWordsJoined, tokenizer)

			if err != nil {
				return nil, err
			}

			if _, ok := searchIndex.IDF[keyWord]; !ok {
				idf, err := bm.IDF(keyWord)
				if err != nil {
					return nil, err
				}
				searchIndex.IDF[keyWord] = idf
			}

			searchIndex.InvertedIndex[keyWord] = append(searchIndex.InvertedIndex[keyWord], DocMeta{
				DocID: data.ID,
				TF:    termFreq,
			})
		}
	}
	searchIndex.DocLengths = bm.DocLengths()
	searchIndex.TotalDocs = bm.CorpusSize()
	searchIndex.AvgDocLength = bm.AvgDocLen()

	return searchIndex, nil
}

func (s *Service) RebuildIndex(ctx context.Context) error {
	if !s.rebuilding.CompareAndSwap(false, true) {
		s.log.Info("rebuild already in progress, skipping")
		return nil
	}
	defer s.rebuilding.Store(false)
	s.log.Info("Starting index rebuild process...")
	comicsData, err := s.db.GetComicsData(ctx)
	if err != nil {
		s.log.Error("", "err", err)
		return err
	}
	newIndex, err := buildSearchIndex(comicsData)
	if err != nil {
		return err
	}

	s.index.Store(newIndex)
	s.log.Info("Index successfully rebuilt")
	return nil
}

func getBM25Score(tf int, idf float64, dl int, avgDL float64) float64 {

	TF := float64(tf)
	DL := float64(dl)
	numerator := TF * (k1 + 1)
	denominator := TF + k1*(1-b+b*(DL/avgDL))

	score := idf * (numerator / denominator)
	return score

}

func getDocScores(normalizedPhrase []string, currentIndex *SearchIndex) map[int]float64 {
	docScores := make(map[int]float64)
	for _, phrase := range normalizedPhrase {
		metaList, ok := currentIndex.InvertedIndex[phrase]
		if !ok {
			continue
		}
		idf := currentIndex.IDF[phrase]
		for _, docMeta := range metaList {
			score := getBM25Score(
				docMeta.TF,
				idf,
				currentIndex.DocLengths[docMeta.DocID-1],
				currentIndex.AvgDocLength,
			)

			docScores[docMeta.DocID] += score
		}
	}
	return docScores
}

func (s *Service) ISearch(ctx context.Context, req ISearchRequest) ([]Comics, error) {
	s.log.Info("Request", "request", req)
	normalizedPhrase, err := s.words.Norm(ctx, req.Phrase)
	if err != nil {
		return nil, err
	}
	currentIndex := s.index.Load()
	if currentIndex == nil {
		return nil, ErrIndexNotReady
	}
	docScores := getDocScores(normalizedPhrase, currentIndex)

	relevantComics := make([]searchResult, 0, len(docScores))
	for docID, docScore := range docScores {
		relevantComics = append(relevantComics, searchResult{
			ID:    docID,
			Score: docScore,
		})
	}

	sort.Slice(relevantComics, func(i, j int) bool {
		return relevantComics[i].Score > relevantComics[j].Score
	})

	if req.Limit > len(relevantComics) {
		req.Limit = len(relevantComics)
	}
	relevantComics = relevantComics[:req.Limit]
	var result []Comics

	for _, comics := range relevantComics {
		if comics.Score == 0 {
			continue
		}

		comicsImageURL, err := s.db.GetImageURL(ctx, comics.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, Comics{
			ID:  int64(comics.ID),
			URL: comicsImageURL,
		})
	}
	return result, nil
}
