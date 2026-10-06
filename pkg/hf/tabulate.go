package hf

import (
	"fmt"
	"strings"
)

// Tabulate formats rows and headers into aligned columnar text.
func Tabulate(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}

	for _, row := range rows {
		for i, val := range row {
			if i < len(colWidths) && len(val) > colWidths[i] {
				colWidths[i] = len(val)
			}
		}
	}

	var sb strings.Builder

	// Header
	for i, h := range headers {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%-*s", colWidths[i], h))
	}
	sb.WriteString("\n")

	// Separator
	for i := range headers {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(strings.Repeat("-", colWidths[i]))
	}
	sb.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, val := range row {
			if i > 0 {
				sb.WriteString(" ")
			}
			// If right-aligned (size)
			if headers[i] == "SIZE ON DISK" {
				sb.WriteString(fmt.Sprintf("%*s", colWidths[i], val))
			} else {
				sb.WriteString(fmt.Sprintf("%-*s", colWidths[i], val))
			}
		}
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}
