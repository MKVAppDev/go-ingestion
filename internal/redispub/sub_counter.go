package redispub

import "fmt"

func (p *Publisher) IncTotalsSub(source, market string) error {
	key := fmt.Sprintf("subs:%s:%s:total", source, market)

	return p.rdb.Incr(p.ctx, key).Err()
}

func (p *Publisher) DecTotalSubs(source, market string) error {
	key := fmt.Sprintf("subs:%s:%s:total", source, market)

	return p.rdb.Decr(p.ctx, key).Err()
}

func (p *Publisher) GetTotalSubs(source, market string) (int64, error) {
	key := fmt.Sprintf("subs:%s:%s:total", source, market)

	return p.rdb.Get(p.ctx, key).Int64()
}
