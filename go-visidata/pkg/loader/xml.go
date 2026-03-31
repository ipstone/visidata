package loader

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

type xmlNode struct {
	Name     string
	Text     string
	Children []*xmlNode
}

func loadXML(path string, reader *os.File) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	root, err := parseXMLTree(reader)
	if err != nil {
		return nil, err
	}

	rows := xmlRows(root)
	return recordsToSheet(path, path, rows), nil
}

func parseXMLTree(reader io.Reader) (*xmlNode, error) {
	decoder := xml.NewDecoder(reader)
	var stack []*xmlNode
	var root *xmlNode

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode xml: %w", err)
		}

		switch typed := token.(type) {
		case xml.StartElement:
			node := &xmlNode{Name: typed.Name.Local}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			text := strings.TrimSpace(string(typed))
			if text == "" {
				continue
			}
			current := stack[len(stack)-1]
			if current.Text == "" {
				current.Text = text
			} else {
				current.Text += " " + text
			}
		}
	}

	if root == nil {
		return nil, fmt.Errorf("xml document is empty")
	}
	return root, nil
}

func xmlRows(root *xmlNode) []any {
	nodes := repeatedXMLNodes(root)
	if len(nodes) == 0 {
		return []any{xmlNodeRecord(root)}
	}
	rows := make([]any, 0, len(nodes))
	for _, node := range nodes {
		rows = append(rows, xmlNodeRecord(node))
	}
	return rows
}

func repeatedXMLNodes(root *xmlNode) []*xmlNode {
	if repeatedChildren(root.Children) {
		return root.Children
	}
	for _, child := range root.Children {
		if repeatedChildren(child.Children) {
			return child.Children
		}
	}
	return nil
}

func repeatedChildren(children []*xmlNode) bool {
	if len(children) < 2 {
		return false
	}
	name := children[0].Name
	for _, child := range children[1:] {
		if child.Name != name {
			return false
		}
	}
	return true
}

func xmlNodeRecord(node *xmlNode) map[string]any {
	record := make(map[string]any)
	if len(node.Children) == 0 {
		record[node.Name] = node.Text
		return record
	}
	for _, child := range node.Children {
		if len(child.Children) == 0 {
			record[child.Name] = child.Text
			continue
		}
		record[child.Name] = xmlNodeRecord(child)
	}
	return record
}
