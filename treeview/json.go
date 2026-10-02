package treeview

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// New parses data as JSON and returns a Model rooted at its
// top-level value. Numbers are shown exactly as written, so an integer
// beyond 2^53 (an ID, say) is not rounded through a float64.
func FromJSON(data []byte) (Model, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return Model{}, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return Model{}, errors.New("treeview: unexpected data after the top-level value")
	}
	return New(valueToNode("root", v)), nil
}

func valueToNode(key string, v interface{}) Node {
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		children := make([]Node, 0, len(keys))
		for _, k := range keys {
			children = append(children, valueToNode(k, val[k]))
		}
		return Node{Label: key + " {}", Children: children}
	case []interface{}:
		children := make([]Node, 0, len(val))
		for i, item := range val {
			children = append(children, valueToNode(strconv.Itoa(i), item))
		}
		return Node{Label: key + " []", Children: children}
	case string:
		return Node{Label: fmt.Sprintf("%s: %q", key, val)}
	case json.Number:
		return Node{Label: key + ": " + val.String()}
	case nil:
		return Node{Label: key + ": null"}
	default:
		return Node{Label: fmt.Sprintf("%s: %v", key, val)}
	}
}
