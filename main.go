package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var version = "0.0.1"

var (
	expr        = flag.String("e", "", "Mermaid diagram string")
	output      = flag.String("o", "", "Save image to file")
	theme       = flag.String("theme", "", "Mermaid theme: dark|default|forest|neutral (auto-detect if empty)")
	width       = flag.String("W", "", "Display width (passed to chafa)")
	height      = flag.String("H", "", "Display height (passed to chafa)")
	background  = flag.String("bg", "transparent", "Mermaid background color")
	showVersion = flag.Bool("version", false, "Print version and exit")
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

	if *showVersion {
		fmt.Printf("mermaidcat %s\n", version)
		os.Exit(0)
	}

	inputs, tmpFiles, err := resolveInput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	logf("TERM_PROGRAM=%s, TERM=%s", os.Getenv("TERM_PROGRAM"), os.Getenv("TERM"))

	// Detect terminal and theme in one probe
	terminal, mermaidTheme := detectAll()
	logf("input files: %v", inputs)
	logf("terminal: %q, theme: %s", terminal, mermaidTheme)

	for i, input := range inputs {
		outPath := ""
		if *output != "" {
			if len(inputs) == 1 {
				outPath = *output
			} else {
				ext := filepath.Ext(*output)
				base := strings.TrimSuffix(*output, ext)
				outPath = fmt.Sprintf("%s-%d%s", base, i+1, ext)
			}
		}
		if err := render(input, mermaidTheme, outPath, terminal); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// resolveInput returns (inputPaths, tmpFilePaths, error).
func resolveInput() ([]string, []string, error) {
	var data []byte

	if *expr != "" {
		data = []byte(*expr)
	} else if flag.NArg() > 0 {
		path := flag.Arg(0)
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("reading file %s: %w", path, err)
		}
	} else {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			var err error
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				return nil, nil, fmt.Errorf("reading stdin: %w", err)
			}
		} else {
			return nil, nil, fmt.Errorf("no input provided. Use: mermaidcat <file>, -e <string>, or pipe via stdin")
		}
	}

	if len(data) == 0 {
		return nil, nil, fmt.Errorf("empty input")
	}

	// Try extracting mermaid blocks from markdown
	blocks := extractMermaidBlocks(string(data))
	if len(blocks) > 0 {
		var paths, tmps []string
		for _, block := range blocks {
			p, t, err := writeTemp([]byte(block))
			if err != nil {
				for _, tmp := range tmps {
					os.Remove(tmp)
				}
				return nil, nil, err
			}
			paths = append(paths, p)
			tmps = append(tmps, t)
		}
		return paths, tmps, nil
	}

	// Treat as raw mermaid input
	p, t, err := writeTemp(data)
	if err != nil {
		return nil, nil, err
	}
	return []string{p}, []string{t}, nil
}

// extractMermaidBlocks extracts content from ```mermaid code blocks.
func extractMermaidBlocks(content string) []string {
	var blocks []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	var current strings.Builder
	inBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inBlock && (trimmed == "```mermaid" || strings.HasPrefix(trimmed, "```mermaid ")) {
			inBlock = true
			current.Reset()
			continue
		}
		if inBlock && trimmed == "```" {
			if current.Len() > 0 {
				blocks = append(blocks, current.String())
			}
			inBlock = false
			continue
		}
		if inBlock {
			current.WriteString(line)
			current.WriteByte('\n')
		}
	}
	return blocks
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
