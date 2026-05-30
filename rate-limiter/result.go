package ratelimiter

import "time"

type Result struct {
	Remaining int
	Allowed bool
	Limit int
	RetryAfter time.Duration
	ResetAt time.Time

}