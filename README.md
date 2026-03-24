# mermaidcat

Render [Mermaid](https://mermaid.js.org/) diagrams directly in your terminal.

mermaidcat pipes Mermaid markup through `mmdc` (mermaid-cli) to produce a PNG, then displays it inline using the best available terminal image protocol. No temporary image files touch disk.

## How It Works

```
Input (file / stdin / -e string)
  -> mmdc renders to PNG (stdout)
    -> iTerm2 IIP or chafa displays inline
```

**Display strategy** (selected automatically):
1. **iTerm2 Inline Images Protocol** — high quality, used when a supported terminal is detected
2. **chafa** — fallback for terminals without IIP support (Kitty Graphics, Sixel, or Unicode block art)

**Terminal detection** mirrors [yazi](https://github.com/sxyazi/yazi)'s approach:
1. Environment variables (`TERM_PROGRAM`, `ITERM_SESSION_ID`, `KITTY_WINDOW_ID`, etc.)
2. tmux saved environment (`tmux show-environment`)
3. CSI escape sequence probe (`XTVERSION` + `DA1`)

## Prerequisites

- [mermaid-cli](https://github.com/mermaid-js/mermaid-cli): `npm install -g @mermaid-js/mermaid-cli`
- [chafa](https://hpjansson.org/chafa/) (optional, only needed if your terminal doesn't support IIP): `brew install chafa` (macOS) / `apt install chafa` (Linux)

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

# From a Markdown file (extracts all ```mermaid blocks)
mermaidcat design.md

# Inline expression
mermaidcat -e "graph LR; A-->B; B-->C"

# Pipe from stdin
cat diagram.mmd | mermaidcat

# Save to file (and display)
mermaidcat diagram.mmd -o output.png

# Save multiple diagrams from Markdown (auto-numbered: output-1.png, output-2.png, ...)
mermaidcat design.md -o output.png

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
| `-W` | Display width (passed to chafa/IIP) | |
| `-H` | Display height (passed to chafa/IIP) | |
| `--bg` | Mermaid background color | `transparent` |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `MERMAIDCAT_TERMINAL` | Force terminal type (e.g. `iterm2`), skipping auto-detection. Useful in environments without a tty (e.g. Claude Code, CI). |
| `MERMAIDCAT_LOG` | Set to `1` or `true` to enable debug logging to stderr. |

## Theme Auto-Detection

When `--theme` is not specified, mermaidcat queries your terminal's background color via [OSC 11](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html) and picks `dark` or `default` accordingly. If detection fails, it falls back to `dark`.

## Terminal Compatibility

Terminals that use **iTerm2 IIP** (high quality, native):

| Terminal | Detection method |
|----------|-----------------|
| iTerm2 | `TERM_PROGRAM`, `ITERM_SESSION_ID`, CSI |
| WezTerm | `TERM_PROGRAM`, `WEZTERM_EXECUTABLE`, CSI |
| Warp | `TERM_PROGRAM`, `WARP_HONOR_PS1` |
| VS Code | `TERM_PROGRAM`, `VSCODE_INJECTION` |
| Tabby | `TERM_PROGRAM`, `TABBY_CONFIG_DIRECTORY` |
| Rio | `TERM_PROGRAM`, `TERM`, CSI |
| Mintty | `TERM_PROGRAM` |
| Hyper | `TERM_PROGRAM` |
| Bobcat | CSI |

Other terminals fall back to **chafa**, which supports Kitty Graphics, Sixel, and Unicode.

## License

MIT
