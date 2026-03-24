package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func render(inputPath, mermaidTheme, outPath, terminal string) error {
	if _, err := exec.LookPath("mmdc"); err != nil {
		return fmt.Errorf("mmdc not found, install via: npm install -g @mermaid-js/mermaid-cli")
	}

	mmdcArgs := []string{
		"-i", inputPath,
		"-o", "-",
		"--outputFormat", "png",
		"-t", mermaidTheme,
		"-b", *background,
		"-w", "3200",
		"-H", "2400",
	}
	mmdcCmd := exec.Command("mmdc", mmdcArgs...)
	mmdcCmd.Stderr = os.Stderr
	logf("mmdc command: mmdc %v", mmdcArgs)

	if isIIPTerminal(terminal) {
		logf("using iTerm2 IIP (terminal=%s)", terminal)
		return renderIIP(mmdcCmd, outPath)
	}
	logf("using chafa (terminal=%q)", terminal)
	return renderChafa(mmdcCmd, outPath)
}

// renderIIP reads mmdc output into memory and displays via iTerm2 IIP.
func renderIIP(mmdcCmd *exec.Cmd, outPath string) error {
	logf("running mmdc...")
	imageData, err := mmdcCmd.Output()
	if err != nil {
		return fmt.Errorf("mmdc failed: %w", err)
	}
	logf("mmdc output: %d bytes PNG", len(imageData))

	if outPath != "" {
		logf("saving to: %s", outPath)
		if err := os.WriteFile(outPath, imageData, 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
	}

	logf("displaying via iTerm2 IIP")
	return displayIIP(imageData)
}

// renderChafa pipes mmdc output to chafa for display.
func renderChafa(mmdcCmd *exec.Cmd, outPath string) error {
	if _, err := exec.LookPath("chafa"); err != nil {
		return fmt.Errorf("chafa not found, install via: brew install chafa (macOS) or apt install chafa (Linux)")
	}

	chafaArgs := buildChafaArgs()
	logf("chafa command: chafa %v", chafaArgs)
	chafaCmd := exec.Command("chafa", chafaArgs...)
	chafaCmd.Stdout = os.Stdout
	chafaCmd.Stderr = os.Stderr

	mmdcOut, err := mmdcCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating mmdc pipe: %w", err)
	}

	var chafaInput io.Reader = mmdcOut

	var outFile *os.File
	if outPath != "" {
		outFile, err = os.Create(outPath)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer outFile.Close()
		chafaInput = io.TeeReader(mmdcOut, outFile)
	}

	chafaCmd.Stdin = chafaInput

	if err := mmdcCmd.Start(); err != nil {
		return fmt.Errorf("starting mmdc: %w", err)
	}
	if err := chafaCmd.Start(); err != nil {
		return fmt.Errorf("starting chafa: %w", err)
	}

	if err := chafaCmd.Wait(); err != nil {
		return fmt.Errorf("chafa failed: %w", err)
	}
	if err := mmdcCmd.Wait(); err != nil {
		return fmt.Errorf("mmdc failed: %w", err)
	}

	return nil
}

func buildChafaArgs() []string {
	var args []string
	if *width != "" && *height != "" {
		args = append(args, "--size", *width+"x"+*height)
	} else if *width != "" {
		args = append(args, "--size", *width)
	}
	args = append(args, "-")
	return args
}
