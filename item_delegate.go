// item_delegate.go contains the itemDelegate struct which is used to render the individual items in the list view.

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/sammcj/gollama/logging"
	"github.com/sammcj/gollama/styles"

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

				// Update the item in the filtered list
				m.SetItem(m.Index(), i)

				// Update the main model list by name match
				for idx, model := range d.appModel.models {
					if model.Name == i.Name {
						d.appModel.models[idx] = i
						break
					}
				}
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

	// If StripString is set in the config, strip it from the model name
	if d.appModel.cfg.StripString != "" {
		model.Name = strings.Replace(model.Name, d.appModel.cfg.StripString, "", 1)
	}

	nameStyle := styles.ItemNameStyle(index)
	dateStyle := styles.ItemDateStyle()
	shaStyle := styles.ItemShaStyle()
	sizeStyle := styles.SizeStyle(model.Size)
	familyStyle := styles.FamilyStyle(model.Family)
	quantStyle := styles.QuantStyle(model.QuantizationLevel)
	modifiedStyle := styles.ItemDateStyle() // Use date style for modified date

	if index == m.Index() {
		// Apply border and highlight styles for selected item
		nameStyle = nameStyle.Bold(true).BorderLeft(true).BorderStyle(lipgloss.InnerHalfBlockBorder()).BorderForeground(styles.GetTheme().GetColour(styles.GetTheme().Colours.ItemBorder)).PaddingLeft(1)
		sizeStyle = sizeStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		quantStyle = quantStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		familyStyle = familyStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		modifiedStyle = modifiedStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		shaStyle = shaStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
		dateStyle = dateStyle.Bold(true).BorderLeft(true).PaddingLeft(-2).PaddingRight(-2)
	}

	// Check if the model is selected in both filtered and unfiltered states
	isSelected := model.Selected
	if d.appModel.list.FilterState() == list.Filtering || d.appModel.list.FilterState() == list.FilterApplied {
		// When filtering, also check the main models list to ensure selection state is accurate
		for _, m := range d.appModel.models {
			if m.Name == model.Name && m.Selected {
				isSelected = true
				break
			}
		}
	}

	if isSelected {
		selectedStyle := styles.SelectedItemStyle()
		// Create new styles that inherit from selected style first
		// Keep the foreground colour but add the background highlight
		nameStyle = selectedStyle.Bold(true).
			Italic(true)
		shaStyle = selectedStyle.Inherit(shaStyle)
		dateStyle = selectedStyle.Inherit(dateStyle)
		sizeStyle = selectedStyle.Inherit(sizeStyle)
		familyStyle = selectedStyle.Inherit(familyStyle)
		quantStyle = selectedStyle.Inherit(quantStyle)
	}

	nameWidth, sizeWidth, quantWidth, modifiedWidth, idWidth, familyWidth := calculateColumnWidths(m.Width())

	// Ensure the text fits within the terminal width
	// Add consistent padding between columns
	padding := 2
	name := nameStyle.Width(nameWidth).Render(truncate(model.Name, nameWidth-padding))
	size := sizeStyle.Width(sizeWidth).Render(fmt.Sprintf("%*.2fGB", sizeWidth-padding-2, model.Size))
	quant := quantStyle.Width(quantWidth).Render(fmt.Sprintf("%-*s", quantWidth-padding, model.QuantizationLevel))
	family := familyStyle.Width(familyWidth).Render(fmt.Sprintf("%-*s", familyWidth-padding, model.Family))
	modified := dateStyle.Width(modifiedWidth).Render(fmt.Sprintf("%-*s", modifiedWidth-padding, model.Modified.Format("2006-01-02")))
	id := shaStyle.Width(idWidth).Render(fmt.Sprintf("%-*s", idWidth-padding, model.ID))

	// Add padding between columns
	spacer := strings.Repeat(" ", padding)
	row := fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s",
		name, spacer, size, spacer, quant, spacer, family, spacer, modified, spacer, id)

	fmt.Fprint(w, row)
}
