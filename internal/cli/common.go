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

func resolveConfigPath(configPath string) (absPath string, configDir string, err error) {
	path := strings.TrimSpace(configPath)
	if path == "" {
		path = "archguard.yaml"
	}
	absPath, err = filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	configDir = filepath.Dir(absPath)
	return absPath, filepath.Clean(configDir), nil
}

func withWorkingDir(dir string, fn func() int) (int, error) {
	original, err := os.Getwd()
	if err != nil {
		return 2, err
	}
	if err := os.Chdir(dir); err != nil {
		return 2, err
	}
	defer func() { _ = os.Chdir(original) }()
	return fn(), nil
}

func filterChangedFiles(allFiles []string, changedOnly bool, changedAgainst string) ([]string, error) {
	against := strings.TrimSpace(changedAgainst)
	if !changedOnly && against == "" {
		return allFiles, nil
	}

	set := map[string]struct{}{}
	commands := make([][]string, 0, 4)
	if changedOnly {
		commands = append(commands,
			[]string{"diff", "--name-only", "--diff-filter=ACMR"},
			[]string{"diff", "--name-only", "--cached", "--diff-filter=ACMR"},
			[]string{"ls-files", "--others", "--exclude-standard"},
		)
	}
	if against != "" {
		commands = append(commands, []string{"diff", "--name-only", "--diff-filter=ACMR", against + "...HEAD"})
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
			return "", fmt.Errorf("git is required for changed-file filtering")
		}
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
