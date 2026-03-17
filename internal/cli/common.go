package cli

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type commonFlags struct {
	configPath string
	format     string
	quiet      bool
	debug      bool
}

func bindCommonFlags(fs *flag.FlagSet, defaults commonFlags) *commonFlags {
	f := defaults
	fs.StringVar(&f.configPath, "config", defaults.configPath, "Path to ArchGuard config")
	fs.StringVar(&f.format, "format", defaults.format, "Output format")
	fs.BoolVar(&f.quiet, "quiet", defaults.quiet, "Suppress non-essential logs")
	fs.BoolVar(&f.debug, "debug", defaults.debug, "Enable debug output")
	return &f
}

func setFlagSetOutput(fs *flag.FlagSet) {
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: archguard %s [flags]\n", fs.Name())
		fs.PrintDefaults()
	}
}

func severityMeetsThreshold(severity, threshold string) bool {
	rank := map[string]int{"warning": 1, "error": 2}
	return rank[severity] >= rank[threshold]
}

func filterChangedFiles(allFiles []string) ([]string, error) {
	set := map[string]struct{}{}
	commands := [][]string{
		{"diff", "--name-only", "--diff-filter=ACMR"},
		{"diff", "--name-only", "--cached", "--diff-filter=ACMR"},
		{"ls-files", "--others", "--exclude-standard"},
	}

	for _, args := range commands {
		out, err := gitOutput(args...)
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			normalized := filepath.ToSlash(filepath.Clean(line))
			set[normalized] = struct{}{}
		}
	}

	result := make([]string, 0)
	for _, file := range allFiles {
		if _, ok := set[file]; ok {
			result = append(result, file)
		}
	}
	sort.Strings(result)
	return result, nil
}

func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("git is required for --changed-only")
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
