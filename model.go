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
}

func (m Model) SelectedStr() string {
  if m.Selected {
    return "X"
  }
  return ""
}

func (m Model) Description() string {
  return fmt.Sprintf("ID: %s, Size: %.2f GB, Quant: %s, Modified: %s", m.ID, m.Size, m.QuantizationLevel, m.Modified.Format("2006-01-02"))
}

func (m Model) FilterValue() string {
  return m.Name
}
