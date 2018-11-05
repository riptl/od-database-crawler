package main

import "fmt"

// https://programming.guide/go/formatting-byte-size-to-human-readable-format.html
func FormatByteCount(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	} else {
		div, exp := int64(unit), 0
		for n := b / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
	}
}
