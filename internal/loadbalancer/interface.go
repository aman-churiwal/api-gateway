package loadbalancer

type Strategy interface {
	// Selects the next target from available targets
	Next(targets []string) string

	// Returns the strategy name
	Name() string
}
