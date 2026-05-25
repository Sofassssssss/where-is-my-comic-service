package nats

import (
	"context"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return &Publisher{}, err
	}
	return &Publisher{
		conn: nc,
	}, nil
}

func (p *Publisher) PublishUpdate(ctx context.Context) error {
	return p.conn.Publish("xkcd.db.updated", []byte("XKCD DB has been updated"))
}

func (p *Publisher) Close() {
	p.conn.Close()
}
