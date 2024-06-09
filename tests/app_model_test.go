// app_model_test.go
package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdate(t *testing.T) {
	m := &AppModel{}
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Errorf("Expected tea.Quit, got %v", cmd)
	}
}
