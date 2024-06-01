package main

import "github.com/charmbracelet/bubbles/key"

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
	SortOrder      string
}

func NewKeyMap() *KeyMap {
	return &KeyMap{
		Space:          key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
		Delete:         key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete selected")),
		SortByName:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "sort name")),
		SortBySize:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort size")),
		SortByModified: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "sort modified")),
		SortByQuant:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "sort quant")),
		SortByFamily:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "sort family")),
		RunModel:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		ConfirmYes:     key.NewBinding(key.WithKeys("y")),
		ConfirmNo:      key.NewBinding(key.WithKeys("n")),
		LinkModel:      key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "link to LMStudio")),
		LinkAllModels:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "link all to LMStudio")),
		ClearScreen:    key.NewBinding(key.WithKeys("c")),
	}
}

// a function to get the state of the sort order
func (k *KeyMap) GetSortOrder() string {
	return k.SortOrder
}
