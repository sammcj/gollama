// item_delegate.go contains the itemDelegate struct which is used to render the individual items in the list view.

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/sammcj/gollama/logging"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type itemDelegate struct {
	appModel *AppModel
}

func NewItemDelegate(appModel *AppModel) itemDelegate {
	return itemDelegate{appModel: appModel}
}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		logging.DebugLogger.Printf("itemDelegate received key: %s\n", msg.String())
		if msg.String() == " " { // space key pressed
			i, ok := m.SelectedItem().(Model)
			if ok {
				logging.DebugLogger.Printf("Delegate toggling selection for model: %s (before: %v)\n", i.Name, i.Selected)
				i.Selected = !i.Selected
				m.SetItem(m.Index(), i)
				// Update the main model list
				d.appModel.models[m.Index()] = i
				logging.DebugLogger.Printf("Updated main model list for model: %s (after: %v)\n", i.Name, i.Selected)
			}
			return nil
		}
	}
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	model, ok := item.(Model)
	if !ok {
		return
	}

	// Alternate colours for model names
	nameColours := []lipgloss.Color{
		lipgloss.Color("#FFFFFF"),
		lipgloss.Color("#818FA1"),
	}

	// If StripString is set in the config, strip it from the model name
	if d.appModel.cfg.StripString != "" {
		model.Name = strings.Replace(model.Name, d.appModel.cfg.StripString, "", 1)
	}

	nameStyle := lipgloss.NewStyle().Foreground(nameColours[index%len(nameColours)])
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("254")).Faint(true)
	sizeStyle := lipgloss.NewStyle().Foreground(sizeColour(model.Size))
	familyStyle := lipgloss.NewStyle().Foreground(familyColour(model.Family, index))
	quantStyle := lipgloss.NewStyle().Foreground(quantColour(model.quantizationLevel))
	modifiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("254"))

	if index == m.Index() {
		// set the name border to pink
		nameStyle = nameStyle.Bold(true).BorderLeft(true).BorderStyle(lipgloss.InnerHalfBlockBorder()).BorderForeground(lipgloss.Color("125")).PaddingLeft(1)
		sizeStyle = sizeStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		quantStyle = quantStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		familyStyle = familyStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		modifiedStyle = modifiedStyle.Foreground(lipgloss.Color("115")).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		idStyle = idStyle.Foreground(lipgloss.Color("225")).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
	}

	if model.Selected {
		// de-indent to allow for selection border
		selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("92")).Bold(true).Italic(true)
		nameStyle = nameStyle.Inherit(selectedStyle)
		idStyle = idStyle.Inherit(selectedStyle)
		sizeStyle = sizeStyle.Inherit(selectedStyle)
		familyStyle = familyStyle.Inherit(selectedStyle)
		quantStyle = quantStyle.Inherit(selectedStyle)
		modifiedStyle = modifiedStyle.Inherit(selectedStyle)
	}

	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth := calculateColumnWidths(m.Width())

	// Ensure the text fits within the terminal width
	name := wrapText(nameStyle.Width(nameWidth).Render(truncate(model.Name, nameWidth)), nameWidth)
	size := wrapText(sizeStyle.Width(sizeWidth).Render(fmt.Sprintf("%.2fGB", model.Size)), sizeWidth)
	quant := wrapText(quantStyle.Width(quantWidth).Render(truncate(model.quantizationLevel, quantWidth)), quantWidth)
	family := wrapText(familyStyle.Width(familyWidth).Render(model.Family), familyWidth)
	modified := wrapText(modifiedStyle.Width(modifiedWidth).Render(model.Modified.Format("2006-01-02")), modifiedWidth)
	id := wrapText(idStyle.Width(idWidth).Render(model.ID), idWidth)

	fmt.Fprint(w, lipgloss.JoinHorizontal(lipgloss.Top, name, size, quant, family, modified, id))
}
