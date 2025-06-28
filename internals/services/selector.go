package services

// BackendSelector implements methods to select an available service
// based on a certain algorithm
type BackendSelector interface {
	SelectService() *Service
}
