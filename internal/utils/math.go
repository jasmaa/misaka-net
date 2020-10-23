package utils

// IntMax finds maximum of two ints
func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// IntMin finds minimum of two ints
func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IntClamp clamps v between a and b inclusive
func IntClamp(v, a, b int) int {
	return IntMax(a, IntMin(v, b))
}
