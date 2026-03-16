package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func render(inputPath, mermaidTheme string) error {
	if _, err := exec.LookPath("mmdc"); err != nil {
		return fmt.Errorf("mmdc not found, install via: npm install -g @mermaid-js/mermaid-cli")
	}
	if _, err := exec.LookPath("chafa"); err != nil {
		return fmt.Errorf("chafa not found, install via: brew install chafa (macOS) or apt install chafa (Linux)")
	}

	mmdcArgs := []string{
		"-i", inputPath,
		"-o", "-",
		"--outputFormat", "png",
		"-t", mermaidTheme,
		"-b", *background,
	}
	mmdcCmd := exec.Command("mmdc", mmdcArgs...)
	mmdcCmd.Stderr = os.Stderr

	chafaArgs := buildChafaArgs()
	chafaCmd := exec.Command("chafa", chafaArgs...)
	chafaCmd.Stdout = os.Stdout
	chafaCmd.Stderr = os.Stderr

	mmdcOut, err := mmdcCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating mmdc pipe: %w", err)
	}

	var chafaInput io.Reader = mmdcOut

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
