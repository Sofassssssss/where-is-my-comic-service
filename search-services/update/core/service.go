package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	publisher   Publisher
	concurrency int
	isBusy      atomic.Int32
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, publisher Publisher, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		publisher:   publisher,
		concurrency: concurrency,
	}, nil
}

func updateWorker(s *Service, jobs <-chan int, insertionsCounter *atomic.Int64, ctx context.Context) {
	for comicsID := range jobs {
		data, err := s.xkcd.Get(ctx, comicsID)
		if err != nil {
			s.log.Error("failed to get xkcd data", "id", comicsID, "error", err)
			continue
		}
		keyWords := strings.Join([]string{
			data.Title,
			data.Alt,
			data.Description,
			data.SafeTitle,
		}, " ")

		/*
			Normalization without removing duplicates to incorporate term frequency metric of BM25 score.
		*/
		normalizedDesc, err := s.words.NormLeaveDuplicates(ctx, keyWords)
		if err != nil {
			s.log.Error("words normalization failed", "id", comicsID, "error", err)
			continue
		}
		err = s.db.Add(ctx, Comics{
			ID:    data.ID,
			URL:   data.URL,
			Words: normalizedDesc,
		})
		if err != nil {
			s.log.Error("failed to add comics to db", "id", comicsID, "error", err)
			continue
		}
		insertionsCounter.Add(1)
	}
}

func getComicsToInsert(comicsAmount int, insertedComicsIDs []int) []int {
	comicsToInsert := make([]int, 0, comicsAmount-len(insertedComicsIDs))
	existing := make(map[int]struct{}, len(insertedComicsIDs))
	for _, id := range insertedComicsIDs {
		existing[id] = struct{}{}
	}
	for i := 1; i <= comicsAmount; i++ {
		if _, ok := existing[i]; !ok {
			comicsToInsert = append(comicsToInsert, i)
		}
	}

	return comicsToInsert
}

func (s *Service) Update(ctx context.Context) (Update, error) {
	if !s.isBusy.CompareAndSwap(0, 1) {
		return Update{}, ErrUpdateRunning
	}
	defer s.isBusy.Store(0)
	comicsAmount, err := s.xkcd.LastID(ctx)
	if err != nil {
		return Update{}, err
	}
	insertedComicsIDs, err := s.db.IDs(ctx)
	if err != nil {
		return Update{}, err
	}
	comicsToInsert := getComicsToInsert(comicsAmount, insertedComicsIDs)
	s.log.Info("Comics to insert", "n", len(comicsToInsert))

	jobs := make(chan int, len(comicsToInsert))
	var insertionsCounter atomic.Int64
	var wg sync.WaitGroup

	for range s.concurrency {
		wg.Go(func() {
			updateWorker(s, jobs, &insertionsCounter, ctx)
		})
	}

	for _, j := range comicsToInsert {
		jobs <- j
	}
	close(jobs)
	wg.Wait()

	err = s.publisher.PublishUpdate(ctx)
	if err != nil {
		s.log.Error("failed to publish event", "err", err)
		return Update{}, err
	}
	return Update{
		ComicsInserted: int(insertionsCounter.Load()),
	}, nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		return ServiceStats{}, err
	}
	comicsTotal, err := s.xkcd.LastID(ctx)
	if err != nil {
		return ServiceStats{}, err
	}
	return ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: comicsTotal,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	if s.isBusy.Load() == 0 {
		return StatusIdle
	}
	return StatusRunning
}

func (s *Service) Drop(ctx context.Context) error {
	err := s.db.Drop(ctx)
	if err != nil {
		return err
	}
	err = s.publisher.PublishUpdate(ctx)
	if err != nil {
		s.log.Error("failed to publish event", "err", err)
		return err
	}
	return nil
}
