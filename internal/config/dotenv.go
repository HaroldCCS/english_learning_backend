package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func loadDotEnv() {
	candidates := findDotEnvCandidates()
	for _, p := range candidates {
		fi, err := os.Stat(p)
		if err != nil || fi.IsDir() {
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" {
				continue
			}
			if os.Getenv(k) != "" {
				continue
			}
			v = strings.Trim(v, "\"'")
			_ = os.Setenv(k, v)
		}
		_ = f.Close()
		return
	}
}

func findDotEnvCandidates() []string {
	wd, err := os.Getwd()
	if err != nil {
		return []string{".env"}
	}

	paths := make([]string, 0, 5)
	cur := wd
	for range 5 {
		paths = append(paths, filepath.Join(cur, ".env"))
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return paths
}
