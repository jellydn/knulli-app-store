package installer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

type xmlNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Text     string
	Children []*xmlNode
}

func updateGamelist(data []byte, menu manifest.Menu, remove bool) ([]byte, error) {
	root := &xmlNode{Name: xml.Name{Local: "gameList"}}
	if len(bytes.TrimSpace(data)) > 0 {
		decoder := xml.NewDecoder(bytes.NewReader(data))
		parsed, err := decodeRoot(decoder)
		if err != nil {
			return nil, fmt.Errorf("parse gamelist: %w", err)
		}
		root = parsed
		if root.Name.Local != "gameList" {
			return nil, fmt.Errorf("parse gamelist: root element must be gameList")
		}
		for {
			_, err := decoder.Token()
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse gamelist: %w", err)
		}
	}
	filtered := root.Children[:0]
	for _, child := range root.Children {
		if child.Name.Local == "game" && childValue(child, "path") == menu.Path {
			continue
		}
		filtered = append(filtered, child)
	}
	root.Children = filtered
	if !remove {
		game := &xmlNode{Name: xml.Name{Local: "game"}}
		game.Children = append(game.Children,
			&xmlNode{Name: xml.Name{Local: "path"}, Text: menu.Path},
			&xmlNode{Name: xml.Name{Local: "name"}, Text: menu.Name},
		)
		if menu.Description != "" {
			game.Children = append(game.Children, &xmlNode{Name: xml.Name{Local: "desc"}, Text: menu.Description})
		}
		root.Children = append(root.Children, game)
	}
	var output bytes.Buffer
	output.WriteString(xml.Header)
	encoder := xml.NewEncoder(&output)
	encoder.Indent("", "  ")
	if err := encodeNode(encoder, root); err != nil {
		return nil, err
	}
	if err := encoder.Flush(); err != nil {
		return nil, err
	}
	output.WriteByte('\n')
	return output.Bytes(), nil
}

func decodeRoot(decoder *xml.Decoder) (*xmlNode, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		if start, ok := token.(xml.StartElement); ok {
			return decodeNode(decoder, start)
		}
	}
}

func decodeNode(decoder *xml.Decoder, start xml.StartElement) (*xmlNode, error) {
	node := &xmlNode{Name: start.Name, Attrs: start.Attr}
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			child, err := decodeNode(decoder, value)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		case xml.CharData:
			if strings.TrimSpace(string(value)) != "" {
				node.Text += string(value)
			}
		case xml.EndElement:
			return node, nil
		}
	}
}

func encodeNode(encoder *xml.Encoder, node *xmlNode) error {
	start := xml.StartElement{Name: node.Name, Attr: node.Attrs}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	if node.Text != "" {
		if err := encoder.EncodeToken(xml.CharData(node.Text)); err != nil {
			return err
		}
	}
	for _, child := range node.Children {
		if err := encodeNode(encoder, child); err != nil {
			return err
		}
	}
	return encoder.EncodeToken(start.End())
}

func childValue(node *xmlNode, name string) string {
	for _, child := range node.Children {
		if child.Name.Local == name {
			return strings.TrimSpace(child.Text)
		}
	}
	return ""
}
