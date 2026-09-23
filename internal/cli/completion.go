package cli

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/core"
)

type pathCompleter struct {
	directoriesOnly bool
	mu              sync.Mutex
	matches         []string
	last            string
	index           int
}

func newPathCompleter(directoriesOnly bool) *pathCompleter {
	return &pathCompleter{directoriesOnly: directoriesOnly, index: -1}
}

func (p *pathCompleter) next(input string) (string, bool) {
	unquoted := strings.Trim(input, "\"'")
	if input == p.last && len(p.matches) > 0 && !(len(p.matches) == 1 && strings.HasSuffix(unquoted, string(os.PathSeparator))) {
		return p.matches[(p.index+1)%len(p.matches)], true
	}
	return "", false
}

// Suggest peeks at the next Tab result without consuming the completion cycle.
func (p *pathCompleter) Suggest(input string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if next, ok := p.next(input); ok {
		return next
	}
	matches, err := pathMatches(input, p.directoriesOnly)
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func (p *pathCompleter) Complete(input string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if next, ok := p.next(input); ok {
		p.index = (p.index + 1) % len(p.matches)
		p.last = next
		return next, nil
	}
	matches, err := pathMatches(input, p.directoriesOnly)
	if err != nil {
		return input, err
	}
	p.matches = matches
	if len(matches) == 0 {
		p.last = input
		return input, nil
	}
	p.index, p.last = 0, matches[0]
	return p.last, nil
}

func pathMatches(input string, directoriesOnly bool) ([]string, error) {
	trimmed := strings.TrimSpace(input)
	quote := ""
	if strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "'") {
		quote = trimmed[:1]
		trimmed = strings.TrimSuffix(trimmed[1:], quote)
	}
	if trimmed == "" {
		trimmed = "." + string(os.PathSeparator)
	}
	path, err := core.ParsePath(trimmed)
	if err != nil {
		return nil, err
	}
	dir, prefix := filepath.Dir(path), filepath.Base(path)
	if strings.HasSuffix(trimmed, `\`) || strings.HasSuffix(trimmed, "/") {
		dir, prefix = path, ""
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// Keep the spelling and separators the user typed, including relative paths.
	parent := trimmed[:strings.LastIndexAny(trimmed, `\/`)+1]
	var matches []string
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || (directoriesOnly && !entry.IsDir()) || !strings.HasPrefix(strings.ToUpper(entry.Name()), strings.ToUpper(prefix)) {
			continue
		}
		name := []rune(entry.Name())
		typed := []rune(prefix)
		if len(typed) > len(name) {
			continue
		}
		match := quote + parent + prefix + string(name[len(typed):])
		if entry.IsDir() {
			match += string(os.PathSeparator)
		}
		matches = append(matches, match+quote)
	}
	sort.Strings(matches)
	return matches, nil
}
