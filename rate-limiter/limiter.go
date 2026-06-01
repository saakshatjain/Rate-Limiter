package ratelimiter

import (
	"time"

	"github.com/saakshatjain/ratelimiter/algorithms"
	"github.com/saakshatjain/ratelimiter/store"
)

type RateLimiter struct {
	config    Config
	store     store.Store
	algorithm algorithms.Algorithm
}

func New(config Config) *RateLimiter {
	s := store.NewMemoryStore()
	algo := &algorithms.TokenBucket{}

	rl := &RateLimiter{
		config:    config,
		store:     s,
		algorithm: algo,
	}

	go func() {
		ticker := time.NewTicker(config.CleanupInterval)
		for range ticker.C {
			s.Cleanup(config.Window * 2)
		}
	}()

	return rl
}

func (rl *RateLimiter) Allow(key string) (Result, error) {
	data, err := rl.store.Get(key)
	if err != nil {
		return Result{}, err
	}

	if data == nil {
		data = &store.ClientData{
			Tokens:      float64(rl.config.Limit),
			LastRequest: time.Now(),
		}
	}

	updatedData, allowed, remaining, retryAfter, resetAt := rl.algorithm.Allow(
		data,
		rl.config.Limit,
		rl.config.Window,
	)
	err = rl.store.Set(key, updatedData, rl.config.Window*2)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		ResetAt:    resetAt,
		Limit:      rl.config.Limit,
	}, nil
}
