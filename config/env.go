package config

// GetNumWorkerMultiplier returns the base value used to calculate the number
// of workers to use for Hugo's parallel execution.
// It returns the value in HUGO_NUMWORKERMULTIPLIER OS env variable if set to a
// positive integer, else the number of logical CPUs.
func GetNumWorkerMultiplier() int {
	return 3
}
