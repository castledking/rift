package main

import (
	"fmt"
	"os"

	"rift/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Get command line arguments
	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	initialModel, initCmd := app.New(path)
	p := tea.NewProgram(
		initialModel,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run init command if any
	if initCmd != nil {
		go func() {
			p.Send(initCmd())
		}()
	}

	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if finalModel, ok := m.(*app.Model); ok && finalModel.HasError() {
		os.Exit(1)
	}
}
