package loader

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadHTML(path string, reader *os.File) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	doc, err := html.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	table := firstHTMLNode(doc, "table")
	if table == nil {
		return nil, fmt.Errorf("html document does not contain a table")
	}

	headers, rows := extractHTMLTable(table)
	if len(headers) == 0 {
		return nil, fmt.Errorf("html table is empty")
	}

	sh := sheet.New(path, path, headers)
	for _, row := range rows {
		if len(row) < len(headers) {
			for len(row) < len(headers) {
				row = append(row, "")
			}
		}
		if len(row) > len(headers) {
			row = row[:len(headers)]
		}
		sh.AddRow(row)
	}
	sh.InferColumnKinds()
	return sh, nil
}

func firstHTMLNode(node *html.Node, tag string) *html.Node {
	if node.Type == html.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := firstHTMLNode(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func extractHTMLTable(table *html.Node) ([]string, [][]string) {
	rows := make([][]string, 0)
	headerRow := -1
	for tr := firstHTMLNode(table, "tr"); tr != nil; tr = nextHTMLNode(tr, "tr") {
		headerCells, rowCells := extractHTMLRow(tr)
		if len(rowCells) == 0 {
			continue
		}
		if headerRow < 0 && headerCells {
			headerRow = len(rows)
		}
		rows = append(rows, rowCells)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if headerRow >= 0 {
		headers := rows[headerRow]
		dataRows := append([][]string{}, rows[:headerRow]...)
		dataRows = append(dataRows, rows[headerRow+1:]...)
		return headers, dataRows
	}
	headers := make([]string, len(rows[0]))
	for i := range headers {
		headers[i] = fmt.Sprintf("col%d", i+1)
	}
	return headers, rows
}

func extractHTMLRow(tr *html.Node) (bool, []string) {
	cells := make([]string, 0)
	hasHeader := false
	for child := tr.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode || (child.Data != "td" && child.Data != "th") {
			continue
		}
		if child.Data == "th" {
			hasHeader = true
		}
		cells = append(cells, strings.TrimSpace(htmlNodeText(child)))
	}
	return hasHeader, cells
}

func htmlNodeText(node *html.Node) string {
	var parts []string
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			text := strings.TrimSpace(current.Data)
			if text != "" {
				parts = append(parts, text)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(parts, " ")
}

func nextHTMLNode(node *html.Node, tag string) *html.Node {
	for current := node.NextSibling; current != nil; current = current.NextSibling {
		if current.Type == html.ElementNode && current.Data == tag {
			return current
		}
		if found := firstHTMLNode(current, tag); found != nil {
			return found
		}
	}
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		for sibling := parent.NextSibling; sibling != nil; sibling = sibling.NextSibling {
			if found := firstHTMLNode(sibling, tag); found != nil {
				return found
			}
		}
	}
	return nil
}
