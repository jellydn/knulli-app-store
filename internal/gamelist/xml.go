package gamelist

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type xmlNode struct {
	Name     xml.Name
	Attrs    []xml.Attr
	Text     string
	Children []*xmlNode
}

// addEntry writes the reviewed entry into its gamelist file inside the
// transaction. An entry with the same path is left untouched: entries this
// installer did not create are never edited.
func addEntry(tx *safefs.Transaction, guard *safefs.Guard, menu manifest.Menu) (bool, error) {
	return editGamelist(tx, guard, menu.Gamelist, func(data []byte) ([]byte, bool, error) {
		return addMenuEntry(data, menu)
	})
}

// replaceEntry rewrites an entry this installer owns — the same gamelist,
// path, name, and description — with updated text.
func replaceEntry(tx *safefs.Transaction, guard *safefs.Guard, previous, menu manifest.Menu) (bool, error) {
	return editGamelist(tx, guard, menu.Gamelist, func(data []byte) ([]byte, bool, error) {
		return replaceOwnedMenuEntry(data, previous, menu)
	})
}

// removeEntry removes an entry this installer owns. Unowned entries with the
// same path are left in place.
func removeEntry(tx *safefs.Transaction, guard *safefs.Guard, menu manifest.Menu) (bool, error) {
	return editGamelist(tx, guard, menu.Gamelist, func(data []byte) ([]byte, bool, error) {
		return removeOwnedMenuEntry(data, menu)
	})
}

// editGamelist rewrites one gamelist file inside the transaction. A missing
// file is parsed as an empty gameList; a plan step that changes nothing
// writes nothing.
func editGamelist(tx *safefs.Transaction, guard *safefs.Guard, gamelist string, change func([]byte) ([]byte, bool, error)) (bool, error) {
	host, err := guard.Resolve(gamelist)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(host)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	updated, changed, err := change(data)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	return true, tx.Write(gamelist, updated, 0644)
}

func addMenuEntry(data []byte, menu manifest.Menu) ([]byte, bool, error) {
	root, err := parseGamelist(data)
	if err != nil {
		return nil, false, err
	}
	for _, child := range root.Children {
		if child.Name.Local == "game" && childValue(child, "path") == menu.Path {
			return data, false, nil
		}
	}
	root.Children = append(root.Children, menuNode(menu))
	output, err := encodeGamelist(root)
	return output, err == nil, err
}

func replaceOwnedMenuEntry(data []byte, oldMenu, newMenu manifest.Menu) ([]byte, bool, error) {
	root, err := parseGamelist(data)
	if err != nil {
		return nil, false, err
	}
	for index, child := range root.Children {
		if menuNodeMatches(child, oldMenu) {
			root.Children[index] = menuNode(newMenu)
			output, encodeErr := encodeGamelist(root)
			return output, encodeErr == nil, encodeErr
		}
	}
	return data, false, nil
}

func removeOwnedMenuEntry(data []byte, menu manifest.Menu) ([]byte, bool, error) {
	root, err := parseGamelist(data)
	if err != nil {
		return nil, false, err
	}
	for index, child := range root.Children {
		if menuNodeMatches(child, menu) {
			root.Children = append(root.Children[:index], root.Children[index+1:]...)
			output, encodeErr := encodeGamelist(root)
			return output, encodeErr == nil, encodeErr
		}
	}
	return data, false, nil
}

func parseGamelist(data []byte) (*xmlNode, error) {
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
	return root, nil
}

func menuNode(menu manifest.Menu) *xmlNode {
	game := &xmlNode{Name: xml.Name{Local: "game"}}
	game.Children = append(game.Children,
		&xmlNode{Name: xml.Name{Local: "path"}, Text: menu.Path},
		&xmlNode{Name: xml.Name{Local: "name"}, Text: menu.Name},
	)
	if menu.Description != "" {
		game.Children = append(game.Children, &xmlNode{Name: xml.Name{Local: "desc"}, Text: menu.Description})
	}
	return game
}

func menuNodeMatches(node *xmlNode, menu manifest.Menu) bool {
	return node.Name.Local == "game" && childValue(node, "path") == menu.Path && childValue(node, "name") == menu.Name && childValue(node, "desc") == menu.Description
}

func encodeGamelist(root *xmlNode) ([]byte, error) {
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
