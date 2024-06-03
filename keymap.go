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
	InspectModel   key.Binding
	Quit           key.Binding
	CopyModel      key.Binding
	PushModel      key.Binding
	Top            key.Binding
	AltScreen      key.Binding
	SortOrder      string
}

func NewKeyMap() *KeyMap {
	return &KeyMap{
		Space:          key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "select")),
		Delete:         key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete selected")),
		InspectModel:   key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),
		Top:            key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "top")),
		SortByName:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "s name")),
		SortBySize:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "s size")),
		SortByModified: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "s modified")),
		SortByQuant:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "s quant size")),
		SortByFamily:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "s family")),
		RunModel:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "run")),
		ConfirmYes:     key.NewBinding(key.WithKeys("y")),
		ConfirmNo:      key.NewBinding(key.WithKeys("n")),
		LinkModel:      key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "link to LMStudio")),
		LinkAllModels:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "link all")),
		CopyModel:      key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "copy model")),
		PushModel:      key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "push model")),
		ClearScreen:    key.NewBinding(key.WithKeys("c")),
		Quit:           key.NewBinding(key.WithKeys("q")),
		AltScreen:      key.NewBinding(key.WithKeys("a")),
	}
}

// a function to get the state of the sort order
func (k *KeyMap) GetSortOrder() string {
	return k.SortOrder
}
