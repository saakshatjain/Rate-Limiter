package algorithms

import (
	"time"

	"github.com/saakshatjain/ratelimiter/store"
)

type Algorithm interface {
    Allow(data *store.ClientData, limit int, window time.Duration) (*store.ClientData, bool, int, time.Duration, time.Time)
}