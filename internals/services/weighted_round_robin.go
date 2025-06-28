package services

import "sync"

type WeightedRRSelector struct {
	services []*Service
	index    int
	counter  int
	mu       sync.Mutex
}

func NewWeightedRoundRobin(services []*Service) *WeightedRRSelector {
	return &WeightedRRSelector{
		services: services,
	}
}

func (wrr *WeightedRRSelector) SelectService() *Service {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.services) == 0 {
		return nil
	}

	for tries := 0; tries < len(wrr.services)*2; tries++ {
		service := wrr.services[wrr.index%len(wrr.services)]

		if service.AliveStatus && wrr.counter < service.Weight {
			wrr.counter++
			return service
		}

		wrr.index++
		wrr.counter = 0
	}

	return nil
}
