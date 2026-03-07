package selector

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// FzfSelector uses the external `fzf` binary for interactive selection.
type FzfSelector struct{}

func (f *FzfSelector) Select(candidates []string, prompt string) (string, error) {
	cmd := exec.Command("fzf", "--prompt="+prompt, "--height=40%", "--reverse", "--no-sort")
	cmd.Stdin = strings.NewReader(strings.Join(candidates, "\n"))

	var out bytes.Buffer
	cmd.Stdout = &out

	// fzf draws its TUI on /dev/tty directly; stderr can be left as-is.
	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 130 {
			return "", ErrCancelled
		}
		return "", fmt.Errorf("fzf: %w", err)
	}

	selected := strings.TrimRight(out.String(), "\n")
	if selected == "" {
		return "", ErrCancelled
	}
	return selected, nil
}

// PecoSelector uses the external `peco` binary.
type PecoSelector struct{}

func (p *PecoSelector) Select(candidates []string, prompt string) (string, error) {
	cmd := exec.Command("peco", "--prompt="+prompt)
	cmd.Stdin = strings.NewReader(strings.Join(candidates, "\n"))

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 130 {
			return "", ErrCancelled
		}
		return "", fmt.Errorf("peco: %w", err)
	}

	selected := strings.TrimRight(out.String(), "\n")
	if selected == "" {
		return "", ErrCancelled
	}
	return selected, nil
}
