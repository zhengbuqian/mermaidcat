# mermaidcat

Render [Mermaid](https://mermaid.js.org/) diagrams directly in your terminal.

mermaidcat pipes Mermaid markup through `mmdc` (mermaid-cli) to produce a PNG, then streams it to `chafa` for inline terminal display — no temporary image files touch disk.

## How It Works

```
Input (file / stdin / -e string)
  → mmdc renders to PNG (stdout)
    → chafa displays inline (stdin)
```

`chafa` auto-detects the best image protocol your terminal supports (Kitty Graphics, iTerm2, Sixel, or Unicode fallback).

## Prerequisites

- [mermaid-cli](https://github.com/mermaid-js/mermaid-cli): `npm install -g @mermaid-js/mermaid-cli`
- [chafa](https://hpjansson.org/chafa/): `brew install chafa` (macOS) / `apt install chafa` (Linux)

## Install

```bash
go install github.com/zhengbuqian/mermaidcat@latest
```

Or build from source:

```bash
git clone https://github.com/zhengbuqian/mermaidcat.git
cd mermaidcat
go build -o mermaidcat .
```

## Usage

```bash
# From a file
mermaidcat diagram.mmd

# Inline expression
mermaidcat -e "graph LR; A-->B; B-->C"

# Pipe from stdin
cat diagram.mmd | mermaidcat

# Save to file (and display)
mermaidcat diagram.mmd -o output.png

# Specify theme
mermaidcat -e "graph TD; Start-->End" --theme forest

# Control display size
mermaidcat diagram.mmd -W 80 -H 40
```

## Options

| Flag | Description | Default |
|------|-------------|---------|
| `-e` | Mermaid diagram string | |
| `-o` | Save rendered image to file | |
| `--theme` | Mermaid theme: `dark`, `default`, `forest`, `neutral` | auto-detect |
| `-W` | Display width (passed to chafa) | |
| `-H` | Display height (passed to chafa) | |
| `--bg` | Mermaid background color | `transparent` |

## Theme Auto-Detection

When `--theme` is not specified, mermaidcat queries your terminal's background color via [OSC 11](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html) and picks `dark` or `default` accordingly. If detection fails, it falls back to `dark`.

## Terminal Compatibility

Since mermaidcat uses `chafa` for rendering, it works with any terminal that `chafa` supports:

| Protocol | Terminals |
|----------|-----------|
| Kitty Graphics | Kitty, Ghostty, WezTerm |
| iTerm2 IIP | iTerm2, WezTerm, Warp |
| Sixel | Windows Terminal, Konsole, foot, VS Code, mlterm |
| Unicode fallback | Any terminal |

## License

MIT
