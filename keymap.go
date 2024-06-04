package main

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyMap struct {
	Space          key.Binding
	Delete         key.Binding
	SortByName     key.Binding
	SortBySize     key.Binding
	SortByModified key.Binding
	SortByQuant    key.Binding
	SortByFamily   key.Binding
	RunModel       key.Binding
	ConfirmYes     key.Binding
	ConfirmNo      key.Binding
	LinkModel      key.Binding
	LinkAllModels  key.Binding
	ClearScreen    key.Binding
	InspectModel   key.Binding
	Quit           key.Binding
	CopyModel      key.Binding
	PushModel      key.Binding
	Top            key.Binding
	AltScreen      key.Binding
	UpdateModel    key.Binding
	Help           key.Binding
	SortOrder      string
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// func newModel() model {
// 	return model{
// 		keys:       *NewKeyMap(),
// 		help:       help.New(),
// 		inputStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),
// 	}
// }

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Space, k.Delete, k.RunModel, k.LinkModel, k.LinkAllModels, k.CopyModel, k.PushModel}, // first column
		{k.SortByName, k.SortBySize, k.SortByModified, k.SortByQuant, k.SortByFamily},           // second column
		{k.Top, k.UpdateModel, k.InspectModel, k.Quit},                                          // third column
	}
}

func NewKeyMap() *KeyMap {
	return &KeyMap{
		Space:          key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
		InspectModel:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),
		Top:            key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "top")),
		RunModel:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		Delete:         key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete")),
		CopyModel:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		PushModel:      key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "push")),
		SortByName:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "^name")),
		SortBySize:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "^size")),
		SortByModified: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "^modified")),
		SortByQuant:    key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "^quant")),
		SortByFamily:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "^family")),
		LinkModel:      key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "link (L=all)")),
		UpdateModel:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update model")),
		LinkAllModels:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "link all")),
		ConfirmYes:     key.NewBinding(key.WithKeys("y")),
		ConfirmNo:      key.NewBinding(key.WithKeys("n")),
		ClearScreen:    key.NewBinding(key.WithKeys("c")),
		Quit:           key.NewBinding(key.WithKeys("q")),
		AltScreen:      key.NewBinding(key.WithKeys("a")),
		Help:           key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
	}
}

func (k *KeyMap) GetSortOrder() string {
	return k.SortOrder
}
