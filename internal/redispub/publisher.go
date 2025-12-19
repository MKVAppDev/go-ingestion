package redispub

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Publisher struct {
	rdb *redis.Client
	ctx context.Context
}

func New(addr string) *Publisher {
	return &Publisher{
		rdb: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
		ctx: context.Background(),
	}
}

func (p *Publisher) Publish(channel string, payload []byte) error {
	return p.rdb.Publish(p.ctx, channel, payload).Err()
}

func (p *Publisher) Close() error {
	return p.rdb.Close()
}

func (p *Publisher) NumSubMany(channels ...string) (map[string]int64, error) {
	res := p.rdb.PubSubNumSub(p.ctx, channels...)
	return res.Result()
}
