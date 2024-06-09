// keymap.go contains the KeyMap struct which is used to define the key bindings for the application.
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

func NewKeyMap() *KeyMap {
	return &KeyMap{
		Space:          key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
		AltScreen:      key.NewBinding(key.WithKeys("A")),
		ClearScreen:    key.NewBinding(key.WithKeys("C")),
		ConfirmNo:      key.NewBinding(key.WithKeys("n")),
		ConfirmYes:     key.NewBinding(key.WithKeys("y")),
		CopyModel:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		Delete:         key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete")),
		Help:           key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "help")),
		InspectModel:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),
		LinkAllModels:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "link all")),
		LinkModel:      key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "link (L=all)")),
		PushModel:      key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "push")),
		Quit:           key.NewBinding(key.WithKeys("q")),
		RunModel:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		SortByFamily:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "^family")),
		SortByModified: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "^modified")),
		SortByName:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "^name")),
		SortByQuant:    key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "^quant")),
		SortBySize:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "^size")),
		Top:            key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "top")),
		UpdateModel:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update model")),
	}
}

func (k *KeyMap) GetSortOrder() string {
	return k.SortOrder
}
