package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jacksonopp/openwaves/cmd/tui/api"
	"github.com/jacksonopp/openwaves/cmd/tui/broadcast"
)

func main() {
	serverURL := envOr("SERVER_URL", "http://localhost:8080")
	adminKey := os.Getenv("ADMIN_KEY")
	scriptPath := envOr("BROADCAST_SCRIPT", "./bin/broadcast.sh")

	client := api.NewClient(serverURL, adminKey)

	var m tea.Model
	if adminKey == "" {
		m = newListenerModel(client, serverURL)
	} else {
		runner := broadcast.New(scriptPath)
		ti := textinput.New()
		ti.CharLimit = 256
		m = model{
			client:      client,
			runner:      runner,
			textInput:   ti,
			currentView: viewList,
			serverURL:   serverURL,
		}
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
