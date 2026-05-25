package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"where-is-my-comic-service/search-services/update/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	query := "INSERT INTO comics (comics_id, image_url, key_words) VALUES ($1, $2, $3)"
	_, err := db.conn.ExecContext(ctx, query, comics.ID, comics.URL, comics.Words)
	return err
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var comicsFetched, wordsTotal, wordsUnique int
	err := db.conn.GetContext(ctx, &comicsFetched, "SELECT COUNT(*) FROM comics")
	if err != nil {
		return core.DBStats{}, err
	}
	err = db.conn.GetContext(ctx, &wordsTotal, "SELECT COALESCE(SUM(cardinality(key_words)), 0) FROM comics")
	if err != nil {
		return core.DBStats{}, err
	}
	err = db.conn.GetContext(ctx, &wordsUnique, "SELECT COALESCE(COUNT(DISTINCT val), 0) FROM comics, unnest(key_words) AS val")
	if err != nil {
		return core.DBStats{}, err
	}
	return core.DBStats{
		ComicsFetched: comicsFetched,
		WordsTotal:    wordsTotal,
		WordsUnique:   wordsUnique,
	}, nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	var comicsIDs []int
	err := db.conn.SelectContext(ctx, &comicsIDs, "SELECT comics_id FROM comics WHERE comics_id > 0 ORDER BY comics_id")
	if err != nil {
		return nil, err
	}
	return comicsIDs, nil
}

func (db *DB) Drop(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM comics")
	if err != nil {
		return err
	}
	return nil
}
