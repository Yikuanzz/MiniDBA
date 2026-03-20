//go:build ignore

// Command check_offline_urls fails if web templates or static assets reference public font CDNs.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	roots := []string{"web/templates", "web/static"}
	bad := false
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			s := string(b)
			if strings.Contains(s, "fonts.googleapis.com") || strings.Contains(s, "fonts.gstatic.com") {
				_, _ = fmt.Fprintf(os.Stderr, "forbidden CDN reference: %s\n", path)
				bad = true
			}
			return nil
		})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "walk %s: %v\n", root, err)
			os.Exit(1)
		}
	}
	if bad {
		os.Exit(1)
	}
}
