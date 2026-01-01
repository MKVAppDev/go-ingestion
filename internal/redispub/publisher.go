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

func (p *Publisher) Ping(ctx context.Context) error {
	return p.rdb.Ping(ctx).Err()
}

func (p *Publisher) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return p.rdb.Subscribe(ctx, channels...)
}
