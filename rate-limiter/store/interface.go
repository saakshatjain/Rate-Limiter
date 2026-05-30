package store
import "time"

type ClientData struct {
	Tokens float64
	LastRequest time.Time
	TimeStamps []time.Time
}

type Store interface {
	Get(key string) (*ClientData, error)
	Set(key string, data *ClientData , ttl time.Duration) error
	Delete(key string) error	
	Cleanup(olderThan time.Duration) error
}
