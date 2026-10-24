package commands

import (
	"fmt"
)

// roundFloat rounds a float to a specified precision
func roundFloat(val float64, precision int) float64 {
	ratio := 1.0
	for i := 0; i < precision; i++ {
		ratio *= 10
	}
	if val < 0 {
		return float64(int(val*ratio-0.5)) / ratio
	}
	return float64(int(val*ratio+0.5)) / ratio
}

// formatNumber formats an integer with comma separators
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d,%03d,%03d", n/1000000, (n/1000)%1000, n%1000)
}

// formatMetricNumber formats an integer with comma separators (alias for formatNumber)
func formatMetricNumber(n int) string {
	return formatNumber(int64(n))
}
