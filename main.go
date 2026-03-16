package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	expr       = flag.String("e", "", "Mermaid diagram string")
	output     = flag.String("o", "", "Save image to file")
	theme      = flag.String("theme", "", "Mermaid theme: dark|default|forest|neutral (auto-detect if empty)")
	width      = flag.String("W", "", "Display width (passed to chafa)")
	height     = flag.String("H", "", "Display height (passed to chafa)")
	background = flag.String("bg", "transparent", "Mermaid background color")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mermaidcat [options] [file]\n\n")
		fmt.Fprintf(os.Stderr, "Render Mermaid diagrams in the terminal.\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  mermaidcat diagram.mmd\n")
		fmt.Fprintf(os.Stderr, "  cat diagram.mmd | mermaidcat\n")
		fmt.Fprintf(os.Stderr, "  mermaidcat -e \"graph LR; A-->B\"\n")
		fmt.Fprintf(os.Stderr, "  mermaidcat diagram.mmd -o output.png\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	input, tmpFile, err := resolveInput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if tmpFile != "" {
		defer os.Remove(tmpFile)
	}

	mermaidTheme := resolveTheme()

	if err := render(input, mermaidTheme); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// resolveInput returns (inputPath, tmpFilePath, error).
// tmpFilePath is non-empty when a temp file was created and needs cleanup.
func resolveInput() (string, string, error) {
	// -e flag: write string to temp file
	if *expr != "" {
		return writeTemp([]byte(*expr))
	}

	// positional argument: use file directly
	if flag.NArg() > 0 {
		path := flag.Arg(0)
		if _, err := os.Stat(path); err != nil {
			return "", "", fmt.Errorf("file not found: %s", path)
		}
		return path, "", nil
	}

	// stdin
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", "", fmt.Errorf("reading stdin: %w", err)
		}
		if len(data) == 0 {
			return "", "", fmt.Errorf("empty input from stdin")
		}
		return writeTemp(data)
	}

	return "", "", fmt.Errorf("no input provided. Use: mermaidcat <file>, -e <string>, or pipe via stdin")
}

func writeTemp(data []byte) (string, string, error) {
	f, err := os.CreateTemp("", "mermaidcat-*.mmd")
	if err != nil {
		return "", "", fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", "", fmt.Errorf("writing temp file: %w", err)
	}
	f.Close()
	return f.Name(), f.Name(), nil
}
