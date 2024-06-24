// top_view.go contains the TopModel struct which is used to render the top view of the application.
package main

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sammcj/gollama/logging"
)

type EditModel struct {
	table    table.Model
	quitting bool
}

func NewEditModel(modelName string) *EditModel {
    logging.DebugLogger.Printf("Initialising model editor\n")

    modelInfos, err := loadModelInfos(modelName)
    if err != nil {
      logging.ErrorLogger.Printf("Failed to load model infos: %v", err)
      return nil
    }

    columns := []table.Column{
      {Title: "Model", Width: 40},
    }

    rows := []table.Row{}
    for _, m := range modelInfos {
      rows = append(rows, table.Row{m.Name})
    }

    // FIXME: At the moment just print the matching model info

    t := table.New(
      table.WithColumns(columns),
      table.WithRows(rows),
      table.WithFocused(true),
      table.WithHeight(7),
    )

    t.SetStyles(table.DefaultStyles())

    // ti := textinput.New()
    // ti.Placeholder = "Edit here"
    // ti.Focus()

    // return &AppModel{
    // 	modelfiles: modelInfos,
    // 	table:      t,
    // 	state:      stateModelList,
    // 	// textInput:  ti,
    // }

    // return AppModel{
    //   modelFiles: modelInfos,
    //   table:      t,
    //   state:      stateModelList,
    // }

    return &EditModel{
      table: t,
    }
}


func (m *EditModel) Init() tea.Cmd {
	logging.DebugLogger.Printf("Initialising edit view\n")
	// return the edit view model
  return nil
}


func (m *EditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  logging.DebugLogger.Printf("edit_view Update called with message type: %T", msg)
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height)

	case tea.Msg:
		return m, nil
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *EditModel) View() string {
	if m.quitting {
    return "Returning to main view...\n"
  }
  return lipgloss.NewStyle().Render(m.table.View())
}


// TODO: New Edit Functions
const (
	ollamaDir       = ".ollama"
	manifestsDir    = "models/manifests"
	blobsDir        = "models/blobs"
	backupManifests = "manifests-backup"
	backupBlobs     = "blobs-backup"
)

func getOllamaDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %v", err)
	}
	return filepath.Join(usr.HomeDir, ollamaDir), nil
}

func loadModelInfos(modelName string) ([]ModelFileInfo, error) {
	log.Println("Loading model infos...")
	var modelFiles []ModelFileInfo

	ollamaPath, err := getOllamaDir()
	if err != nil {
		return nil, err
	}

	manifestsPath := filepath.Join(ollamaPath, manifestsDir)

	err = filepath.Walk(manifestsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(manifestsPath, path)
			if err != nil {
				return err
			}
			modelFileName := strings.ReplaceAll(relPath, string(filepath.Separator), "/")
			modelInfo := ModelFileInfo{
				Name:         modelFileName,
				ManifestPath: path,
			}
			modelFiles = append(modelFiles, modelInfo)
			logging.InfoLogger.Printf("Found model: %s\n", modelFileName)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %v", manifestsPath, err)
	}

	if len(modelFiles) == 0 {
		return nil, fmt.Errorf("no models found in %s", manifestsPath)
	}

	// Find the modelfile for the selected model
	for _, model := range modelFiles {
		if model.Name == modelName {
			return []ModelFileInfo{model}, nil
		}
	}

	return modelFiles, nil
}

