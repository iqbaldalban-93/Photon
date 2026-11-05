package backend

import "app/internal/concurrency"

type AdaptiveConcurrency = concurrency.AdaptiveConcurrency

func NewAdaptiveConcurrency(maxWorkers int, onLimitChanged func(limit, max int)) *AdaptiveConcurrency {
	return concurrency.NewAdaptiveConcurrency(maxWorkers, onLimitChanged)
}
