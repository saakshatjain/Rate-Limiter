package algorithms

import (
	"time"
	"github.com/saakshatjain/ratelimiter/store"
)

type TokenBucket struct{}

func (tb *TokenBucket) Allow(data *store.ClientData, limit int, window time.Duration) (*store.ClientData, bool, int, time.Duration, time.Time) {
	now := time.Now()
	elapsed := now.Sub(data.LastRequest) 

	refillRate   := float64(limit) / window.Seconds()
    tokensToAdd  := elapsed.Seconds() * refillRate
    data.Tokens  += tokensToAdd
	if data.Tokens > float64(limit) {
		data.Tokens = float64(limit)
	}

	data.LastRequest = now

	if (data.Tokens>=1) {
		data.Tokens -= 1
		return data, true, int(data.Tokens), 0, now.Add(window)
	}
	retryAfter  := time.Duration(float64(time.Second) / refillRate)

	return data, false, int(data.Tokens), retryAfter, now.Add(window)
}