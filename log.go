package main

import (
	"fmt"
	"os"
	"strings"
)

func debugLog() bool {
	v := os.Getenv("MERMAIDCAT_LOG")
	return v == "1" || strings.EqualFold(v, "true")
}

func logf(format string, args ...any) {
	if debugLog() {
		fmt.Fprintf(os.Stderr, "[mermaidcat] "+format+"\n", args...)
	}
}
