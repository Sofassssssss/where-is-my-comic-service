package nats

import "github.com/nats-io/nats.go"

type Subscriber struct {
	conn *nats.Conn
}

func NewSubscriber(url string) (*Subscriber, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return &Subscriber{}, err
	}
	return &Subscriber{
		conn: nc,
	}, nil
}

func (s *Subscriber) Subscribe(handler func()) (*nats.Subscription, error) {
	sub, err := s.conn.Subscribe("xkcd.db.updated", func(msg *nats.Msg) {
		handler()
	})
	if err != nil {
		return &nats.Subscription{}, err
	}
	return sub, nil
}

func (s *Subscriber) Close() {
	s.conn.Close()
}
