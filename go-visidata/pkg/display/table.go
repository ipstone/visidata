package display

import (
	"fmt"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
	"github.com/mattn/go-runewidth"
)

func RenderPreview(sh *sheet.Sheet, limit int) string {
	if limit <= 0 {
		limit = 10
	}

	widths := columnWidths(sh, limit)
	var b strings.Builder
	b.WriteString(sh.Summary())
	b.WriteString("\n")

	for i, col := range sh.Columns {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(padRight(fmt.Sprintf("%s <%s>", col.Name, col.Kind), widths[i]))
	}
	b.WriteString("\n")

	for i, width := range widths {
		if i > 0 {
			b.WriteString("-+-")
		}
		b.WriteString(strings.Repeat("-", width))
	}
	b.WriteString("\n")

	rows := sh.Rows
	if len(rows) > limit {
		rows = rows[:limit]
	}
	for _, row := range rows {
		for i := range sh.Columns {
			if i > 0 {
				b.WriteString(" | ")
			}
			value := ""
			if i < len(row) {
				value = row[i]
			}
			b.WriteString(padRight(value, widths[i]))
		}
		b.WriteString("\n")
	}

	if len(sh.Rows) > limit {
		b.WriteString(fmt.Sprintf("... showing first %d of %d row(s)\n", limit, len(sh.Rows)))
	}

	return b.String()
}

func columnWidths(sh *sheet.Sheet, limit int) []int {
	widths := make([]int, len(sh.Columns))
	for i, col := range sh.Columns {
		widths[i] = displayWidth(fmt.Sprintf("%s <%s>", col.Name, col.Kind))
	}

	rows := sh.Rows
	if len(rows) > limit {
		rows = rows[:limit]
	}
	for _, row := range rows {
		for i := range sh.Columns {
			if i < len(row) && displayWidth(row[i]) > widths[i] {
				widths[i] = displayWidth(row[i])
			}
		}
	}
	return widths
}

func padRight(value string, width int) string {
	if displayWidth(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-displayWidth(value))
}

func displayWidth(value string) int {
	return runewidth.StringWidth(value)
}
