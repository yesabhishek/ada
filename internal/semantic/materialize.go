package semantic

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/yesabhishek/ada/internal/language"
)

type Materializer struct {
	registry *language.Registry
}

func NewMaterializer(registry *language.Registry) *Materializer {
	return &Materializer{registry: registry}
}

type Edit struct {
	Path        string
	StartByte   int
	EndByte     int
	Replacement []byte
}

func (m *Materializer) ApplyEdits(ctx context.Context, path string, source []byte, edits []Edit) ([]byte, error) {
	if len(edits) == 0 {
		return append([]byte(nil), source...), nil
	}
	sort.Slice(edits, func(i, j int) bool {
		return edits[i].StartByte > edits[j].StartByte
	})
	out := append([]byte(nil), source...)
	lastStart := len(out) + 1
	for _, edit := range edits {
		if edit.StartByte < 0 || edit.EndByte > len(out) || edit.StartByte > edit.EndByte {
			return nil, fmt.Errorf("invalid edit range %d:%d for %s", edit.StartByte, edit.EndByte, path)
		}
		if edit.EndByte > lastStart {
			return nil, fmt.Errorf("overlapping semantic edits for %s", path)
		}
		lastStart = edit.StartByte
		out = bytes.Join([][]byte{out[:edit.StartByte], edit.Replacement, out[edit.EndByte:]}, nil)
	}
	adapter := m.registry.ForPath(path)
	if adapter == nil {
		return out, nil
	}
	formatted, err := adapter.Format(ctx, path, out)
	if err != nil {
		return nil, fmt.Errorf("format materialized file %s: %w", path, err)
	}
	return formatted, nil
}
