package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sammcj/gollama/styles"
)

// TODO: Complete the progress bar and implement in the operations

const (
	padding  = 2
	maxWidth = 80
)

func (m progressModel) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m progressModel) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + styles.HelpTextStyle().Render("Press any key to quit")
}

type tickMsg time.Time

type progressModel struct {
	progress progress.Model
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.progress.Percent() == 1.0 {
			return m, tea.Quit
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		cmd := m.progress.IncrPercent(0.25)
		return m, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m progressModel) ViewProgress() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		pad + styles.HelpTextStyle().Render("Press any key to quit")
}

// A progress demo function that shows a progress bar that updates 25% every second
func progressDemo() {
	// tickCmd() is a helper function that returns a Cmd that sends a tickMsg every second
	// to the update function. This is how we animate the progress bar.
	tickCmd := tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})

	p := progress.New(progress.WithDefaultGradient())
	m := progressModel{progress: p}

	// Start the progress bar
	t := tea.NewProgram(m)

	if err := func() error {
		_, err := t.Run()
		return err
	}(); err != nil {
		panic(err)
	}

	// Update the progress bar with tickMsg
	tickCmd()

	// Stop the progress bar
	t.Kill()
}
