// model.go contains the Model struct which is used to represent the data for each item in the list view.
package main

import (
	"fmt"
	"time"
)

type Model struct {
	Name              string
	ID                string
	Size              float64
	QuantizationLevel string
	Modified          time.Time
	Selected          bool
	Family            string
	ParameterSize     string
}

func (m Model) SelectedStr() string {
	if m.Selected {
		return "X"
	}
	return ""
}

func (m Model) Description() string {
	paramSizeStr := ""
	if m.ParameterSize != "" {
		paramSizeStr = fmt.Sprintf(", Parameters: %s", m.ParameterSize)
	}
	return fmt.Sprintf("ID: %s, Size: %.2f GB, Quant: %s%s, Modified: %s", m.ID, m.Size, m.QuantizationLevel, paramSizeStr, m.Modified.Format("2006-01-02"))
}

func (m Model) FilterValue() string {
	return m.Name
}
