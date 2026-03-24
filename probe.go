package main

import (
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// probeResult holds the results of a single tty probe.
type probeResult struct {
	terminal string // detected terminal brand
	isDark   bool   // true if dark background
	probed   bool   // true if background was successfully probed
}

// probeTTY sends escape sequence queries and parses the response.
// Mirrors yazi's approach: write to stdout, read from stdin, using poll for non-blocking reads.
func probeTTY() probeResult {
	result := probeResult{}

	// Get stdin fd - check if it's a tty
	stdinFd := int(os.Stdin.Fd())
	stdoutFd := int(os.Stdout.Fd())

	// If stdin is not a tty, try /dev/tty
	if !isatty(stdinFd) {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			logf("probe: cannot open /dev/tty: %v", err)
			return result
		}
		defer tty.Close()
		stdinFd = int(tty.Fd())
		stdoutFd = int(tty.Fd())
	}

	// Set raw mode on stdin
	oldState, err := makeRawFd(stdinFd)
	if err != nil {
		logf("probe: cannot set raw mode: %v", err)
		return result
	}
	defer restoreTerminalFd(stdinFd, oldState)

	// Send all queries at once to stdout (like yazi does):
	// 1. XTVERSION: \x1b[>q
	// 2. OSC 11 background: \x1b]11;?\x07
	// 3. DA1 sentinel: \x1b[0c
	query := "\x1b[>q\x1b]11;?\x07\x1b[0c"
	unix.Write(stdoutFd, []byte(query))

	// Read response byte by byte with poll, until DA1 response (ends with 'c' after \x1b[?)
	resp := readUntilDA1(stdinFd, 1000*time.Millisecond)
	logf("probe: raw response (%d bytes): %q", len(resp), resp)

	if len(resp) == 0 {
		logf("probe: no response from terminal")
		return result
	}

	// Parse terminal brand from response
	result.terminal = brandFromCSI(resp)

	// Parse background color from OSC 11 response
	bgTheme := parseOSC11Response(resp)
	if bgTheme != "" {
		result.probed = true
		result.isDark = (bgTheme == "dark")
	}

	return result
}

// isatty checks if fd is a terminal.
func isatty(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	return err == nil
}

// makeRawFd sets the terminal to raw mode and returns the old state.
func makeRawFd(fd int) (*unix.Termios, error) {
	old, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	raw := *old
	raw.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	raw.Oflag &^= unix.OPOST
	raw.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Cflag &^= unix.CSIZE | unix.PARENB
	raw.Cflag |= unix.CS8
	raw.Cc[unix.VMIN] = 0
	raw.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, &raw); err != nil {
		return nil, err
	}
	return old, nil
}

// restoreTerminalFd restores terminal settings.
func restoreTerminalFd(fd int, state *unix.Termios) {
	unix.IoctlSetTermios(fd, unix.TCSETS, state)
}

// readUntilDA1 reads from fd byte by byte using poll(), until DA1 response is detected.
// DA1 response: \x1b[?...c
// Mirrors yazi's read_until_da1().
func readUntilDA1(fd int, timeout time.Duration) string {
	var buf []byte
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		// Poll for data availability
		pollFds := []unix.PollFd{
			{Fd: int32(fd), Events: unix.POLLIN},
		}
		pollTimeout := int(remaining.Milliseconds())
		if pollTimeout > 30 {
			pollTimeout = 30
		}
		n, err := unix.Poll(pollFds, pollTimeout)
		if err != nil || n == 0 {
			continue
		}

		// Read one byte
		var b [1]byte
		n2, err := unix.Read(fd, b[:])
		if err != nil || n2 == 0 {
			continue
		}
		buf = append(buf, b[0])

		// Check for DA1 response: ends with 'c' and contains \x1b[?
		if b[0] == 'c' && len(buf) >= 4 {
			s := unsafe.String(unsafe.SliceData(buf), len(buf))
			// Find last \x1b in buffer
			lastEsc := strings.LastIndex(s, "\x1b")
			if lastEsc >= 0 && lastEsc+1 < len(s) && s[lastEsc+1] == '[' {
				// Check if it starts with \x1b[?
				rest := s[lastEsc:]
				if strings.HasPrefix(rest, "\x1b[?") {
					break
				}
			}
		}
	}

	return string(buf)
}

// brandFromCSI extracts terminal brand from CSI response (XTVERSION + DA1).
// Matches yazi's Brand::from_csi() — uses substring matching.
func brandFromCSI(resp string) string {
	checks := []struct {
		substr string
		brand  string
	}{
		{"kitty", BrandKitty},
		{"Konsole", BrandKonsole},
		{"iTerm2", BrandIterm2},
		{"WezTerm", BrandWezTerm},
		{"foot", BrandFoot},
		{"ghostty", BrandGhostty},
		{"Warp", BrandWarp},
		{"tmux ", BrandTmux},
		{"libvterm", BrandVTerm},
		{"Bobcat", BrandBobcat},
	}
	for _, c := range checks {
		if strings.Contains(resp, c.substr) {
			return c.brand
		}
	}
	return ""
}
