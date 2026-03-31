package tui

import (
	"sort"
	"strings"
)

type paletteMatch struct {
	command commandInfo
	score   int
}

func (a *App) commandPaletteMatches() []commandInfo {
	query := strings.TrimSpace(strings.ToLower(string(a.inputValue)))
	if query == "" {
		return append([]commandInfo(nil), commandCatalog...)
	}

	matches := make([]paletteMatch, 0, len(commandCatalog))
	for _, command := range commandCatalog {
		score, ok := commandPaletteScore(command, query)
		if !ok {
			continue
		}
		matches = append(matches, paletteMatch{command: command, score: score})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].command.Name < matches[j].command.Name
	})

	commands := make([]commandInfo, len(matches))
	for i, match := range matches {
		commands[i] = match.command
	}
	return commands
}

func commandPaletteScore(command commandInfo, query string) (int, bool) {
	candidates := []string{
		strings.ToLower(command.Name),
		strings.ToLower(command.Keys),
		strings.ToLower(command.Help),
	}

	best := -1
	for _, candidate := range candidates {
		score, ok := fuzzyScore(candidate, query)
		if ok && score > best {
			best = score
		}
	}
	return best, best >= 0
}

func fuzzyScore(candidate, query string) (int, bool) {
	if candidate == "" {
		return 0, false
	}
	if strings.Contains(candidate, query) {
		score := 100 - strings.Index(candidate, query)
		if strings.HasPrefix(candidate, query) {
			score += 25
		}
		return score, true
	}

	score := 0
	last := -1
	for _, needle := range query {
		index := strings.IndexRune(candidate[last+1:], needle)
		if index < 0 {
			return 0, false
		}
		actual := last + 1 + index
		score++
		if actual == last+1 {
			score += 4
		}
		if actual == 0 {
			score += 8
		}
		last = actual
	}
	return score, true
}

func (a *App) paletteSelection(matches []commandInfo) int {
	if len(matches) == 0 {
		return 0
	}
	if a.paletteIndex < 0 {
		return 0
	}
	if a.paletteIndex >= len(matches) {
		return len(matches) - 1
	}
	return a.paletteIndex
}
