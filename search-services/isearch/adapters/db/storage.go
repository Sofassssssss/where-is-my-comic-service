package db

import (
	"context"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"where-is-my-comic-service/search-services/isearch/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

type comicsKeyWords struct {
	ComicsID int            `db:"comics_id"`
	KeyWords pq.StringArray `db:"key_words"`
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

func (db *DB) GetComicsData(ctx context.Context) ([]core.ComicsKeyWords, error) {
	var comicsData []comicsKeyWords
	err := db.conn.SelectContext(ctx, &comicsData, "SELECT comics_id, key_words FROM comics")

	if err != nil {
		return nil, err
	}
	result := make([]core.ComicsKeyWords, len(comicsData))

	for i, row := range comicsData {
		result[i] = core.ComicsKeyWords{
			ID:       row.ComicsID,
			KeyWords: row.KeyWords,
		}
	}
	return result, nil
}

func (db *DB) GetImageURL(ctx context.Context, comicsID int) (string, error) {
	var imageURL string
	query := `SELECT image_url FROM comics WHERE comics_id = $1`
	err := db.conn.QueryRowContext(ctx, query, comicsID).Scan(&imageURL)
	if err != nil {
		return "", err
	}
	return imageURL, nil
}
