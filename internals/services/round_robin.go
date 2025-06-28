package services

import "sync"

type RoundRobinSelector struct {
	services []*Service
	index    int
	mu       sync.Mutex
}

func NewRoundRobinSelector(services []*Service) *RoundRobinSelector {
	return &RoundRobinSelector{
		services: services,
	}
}

func (rr *RoundRobinSelector) SelectService() *Service {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.services) == 0 {
		return nil
	}

	for i := 0; i < len(rr.services); i++ {
		svc := rr.services[rr.index%len(rr.services)]
		rr.index++

		if svc.AliveStatus {
			return svc
		}
	}

	return nil // No healthy service were encountered
}
