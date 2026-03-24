package main

import "fmt"

// parseOSC11Response parses "\033]11;rgb:RRRR/GGGG/BBBB\033\\" and returns
// "dark" or "default" based on perceived brightness.
func parseOSC11Response(resp string) string {
	var r, g, b uint32
	for i := 0; i < len(resp)-4; i++ {
		if resp[i] == 'r' && resp[i+1] == 'g' && resp[i+2] == 'b' && resp[i+3] == ':' {
			_, err := fmt.Sscanf(resp[i:], "rgb:%04x/%04x/%04x", &r, &g, &b)
			if err != nil {
				_, err = fmt.Sscanf(resp[i:], "rgb:%02x/%02x/%02x", &r, &g, &b)
				if err != nil {
					return ""
				}
				r, g, b = r*257, g*257, b*257
			}
			brightness := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
			if brightness < 0.5 {
				return "dark"
			}
			return "default"
		}
	}
	return ""
}
