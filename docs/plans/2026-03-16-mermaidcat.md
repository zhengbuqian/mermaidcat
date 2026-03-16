# mermaidcat 实现计划

**目标：** Go CLI 工具，将 Mermaid 图通过 mmdc 渲染为 PNG，通过 chafa 在终端内联显示

**架构：** 三层管道 — 输入处理 → mmdc 渲染(stdout) → chafa 显示(stdin)。主题通过 OSC 11 自动检测终端背景色。保存通过 TeeReader 分流。

**技术栈：** Go 标准库（flag, os/exec, io），外部依赖 mmdc + chafa

---

### 任务 1: 项目初始化 + 参数解析 (main.go)

**文件：**
- 创建: `main.go`
- 创建: `go.mod`

**步骤 1: 初始化 Go module**

运行: `go mod init github.com/buqian/mermaidcat`

**步骤 2: 实现 main.go**

```go
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
```

**步骤 3: 验证编译**

运行: `go build -o /dev/null .`
预期: 编译失败（resolveTheme 和 render 未定义），但语法正确

---

### 任务 2: 终端主题检测 (theme.go)

**文件：**
- 创建: `theme.go`

**步骤 1: 实现 theme.go**

```go
package main

import (
	"fmt"
	"os"
	"time"
)

// resolveTheme returns the mermaid theme to use.
// Priority: --theme flag > auto-detect via OSC 11 > fallback "dark".
func resolveTheme() string {
	if *theme != "" {
		return *theme
	}
	if detected := detectTerminalBackground(); detected != "" {
		return detected
	}
	return "dark"
}

// detectTerminalBackground queries the terminal background color via OSC 11
// and returns "dark" or "default" based on perceived brightness.
func detectTerminalBackground() string {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return ""
	}
	defer tty.Close()

	// Save terminal state and set raw mode for reading response
	oldState, err := makeRaw(tty)
	if err != nil {
		return ""
	}
	defer restoreTerminal(tty, oldState)

	// Send OSC 11 query: \033]11;?\033\\
	fmt.Fprint(tty, "\033]11;?\033\\")

	// Read response with timeout
	result := make(chan string, 1)
	go func() {
		buf := make([]byte, 256)
		n, err := tty.Read(buf)
		if err != nil || n == 0 {
			result <- ""
			return
		}
		result <- string(buf[:n])
	}()

	select {
	case resp := <-result:
		return parseOSC11Response(resp)
	case <-time.After(200 * time.Millisecond):
		return ""
	}
}

// parseOSC11Response parses "\033]11;rgb:RRRR/GGGG/BBBB\033\\" and returns
// "dark" or "default" based on perceived brightness.
func parseOSC11Response(resp string) string {
	// Look for rgb: pattern
	var r, g, b uint32
	for i := 0; i < len(resp)-4; i++ {
		if resp[i] == 'r' && resp[i+1] == 'g' && resp[i+2] == 'b' && resp[i+3] == ':' {
			_, err := fmt.Sscanf(resp[i:], "rgb:%04x/%04x/%04x", &r, &g, &b)
			if err != nil {
				// Try 2-digit hex format
				_, err = fmt.Sscanf(resp[i:], "rgb:%02x/%02x/%02x", &r, &g, &b)
				if err != nil {
					return ""
				}
				// Scale to 16-bit
				r, g, b = r*257, g*257, b*257
			}
			// Calculate perceived brightness (ITU-R BT.601)
			brightness := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
			if brightness < 0.5 {
				return "dark"
			}
			return "default"
		}
	}
	return ""
}
```

**步骤 2: 验证编译**

运行: `go build -o /dev/null .`
预期: 编译失败（render, makeRaw, restoreTerminal 未定义）

---

### 任务 3: 终端 raw mode (term.go)

**文件：**
- 创建: `term.go`

**步骤 1: 实现 term.go**

使用 golang.org/x/term 来处理终端 raw mode：

运行: `go get golang.org/x/term`

```go
package main

import (
	"os"

	"golang.org/x/term"
)

func makeRaw(tty *os.File) (*term.State, error) {
	return term.MakeRaw(int(tty.Fd()))
}

func restoreTerminal(tty *os.File, state *term.State) {
	term.Restore(int(tty.Fd()), state)
}
```

---

### 任务 4: 渲染管道 (render.go)

**文件：**
- 创建: `render.go`

**步骤 1: 实现 render.go**

```go
package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func render(inputPath, mermaidTheme string) error {
	// Check dependencies
	if _, err := exec.LookPath("mmdc"); err != nil {
		return fmt.Errorf("mmdc not found, install via: npm install -g @mermaid-js/mermaid-cli")
	}
	if _, err := exec.LookPath("chafa"); err != nil {
		return fmt.Errorf("chafa not found, install via: brew install chafa (macOS) or apt install chafa (Linux)")
	}

	// Build mmdc command
	mmdcArgs := []string{
		"-i", inputPath,
		"-o", "-",
		"--outputFormat", "png",
		"-t", mermaidTheme,
		"-b", *background,
	}
	mmdcCmd := exec.Command("mmdc", mmdcArgs...)
	mmdcCmd.Stderr = os.Stderr

	// Build chafa command
	chafaArgs := buildChafaArgs()
	chafaCmd := exec.Command("chafa", chafaArgs...)
	chafaCmd.Stdout = os.Stdout
	chafaCmd.Stderr = os.Stderr

	// Connect mmdc stdout -> chafa stdin, with optional tee to file
	mmdcOut, err := mmdcCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating mmdc pipe: %w", err)
	}

	var chafaInput io.Reader = mmdcOut

	// If -o specified, tee the PNG data to file
	var outFile *os.File
	if *output != "" {
		outFile, err = os.Create(*output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer outFile.Close()
		chafaInput = io.TeeReader(mmdcOut, outFile)
	}

	chafaCmd.Stdin = chafaInput

	// Start both commands
	if err := mmdcCmd.Start(); err != nil {
		return fmt.Errorf("starting mmdc: %w", err)
	}
	if err := chafaCmd.Start(); err != nil {
		return fmt.Errorf("starting chafa: %w", err)
	}

	// Wait for chafa first (it's the consumer), then mmdc
	if err := chafaCmd.Wait(); err != nil {
		return fmt.Errorf("chafa failed: %w", err)
	}
	if err := mmdcCmd.Wait(); err != nil {
		return fmt.Errorf("mmdc failed: %w", err)
	}

	return nil
}

func buildChafaArgs() []string {
	args := []string{}
	if *width != "" && *height != "" {
		args = append(args, "--size", *width+"x"+*height)
	} else if *width != "" {
		args = append(args, "--size", *width)
	}
	args = append(args, "-")
	return args
}
```

**步骤 2: 验证完整编译**

运行: `go build -o mermaidcat .`
预期: 编译成功

---

### 任务 5: 端到端测试

**步骤 1: 安装依赖**

运行:
```bash
npm install -g @mermaid-js/mermaid-cli
# macOS: brew install chafa
# Linux: apt install chafa
```

**步骤 2: 测试文件输入**

```bash
echo 'graph LR; A-->B; B-->C' > /tmp/test.mmd
./mermaidcat /tmp/test.mmd
```
预期: 终端内显示流程图

**步骤 3: 测试 -e 输入**

```bash
./mermaidcat -e "graph TD; Start-->End"
```
预期: 终端内显示流程图

**步骤 4: 测试 stdin**

```bash
echo 'graph LR; X-->Y' | ./mermaidcat
```
预期: 终端内显示流程图

**步骤 5: 测试保存**

```bash
./mermaidcat -e "graph LR; A-->B" -o test.png
file test.png
```
预期: 终端显示图片，且 test.png 为有效 PNG 文件

**步骤 6: 测试主题**

```bash
./mermaidcat -e "graph LR; A-->B" --theme forest
```
预期: 使用 forest 主题渲染

---

### 任务 6: 提交

**步骤 1: 提交**

```bash
git add main.go render.go theme.go term.go go.mod go.sum
git commit -s -m "feat: initial mermaidcat implementation

Mermaid diagram renderer for terminal using mmdc + chafa pipeline.
Supports file, stdin, and -e string input.
Auto-detects terminal theme via OSC 11.
Optional image save with -o flag."
```
