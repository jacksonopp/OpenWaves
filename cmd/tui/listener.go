package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jacksonopp/openwaves/cmd/tui/api"
	"github.com/jacksonopp/openwaves/cmd/tui/player"
)

// listenerModel is the read-only TUI for listeners (no admin key).
type listenerModel struct {
	client      *api.Client
	pl          *player.Player
	stations    []api.PublicStation
	selected    int
	playingURL  string
	statusMsg   string
	err         error
	width       int
	height      int
	serverURL   string
}

type listenerStationsMsg []api.PublicStation

func newListenerModel(client *api.Client, serverURL string) listenerModel {
	return listenerModel{
		client:    client,
		pl:        player.New(),
		serverURL: serverURL,
	}
}

func (m listenerModel) Init() tea.Cmd {
	return tea.Batch(listenerFetchStations(m.client), listenerTickCmd())
}

func (m listenerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		cmds = append(cmds, listenerFetchStations(m.client), listenerTickCmd())

	case listenerStationsMsg:
		// preserve selection
		m.stations = []api.PublicStation(msg)
		sort.Slice(m.stations, func(i, j int) bool {
			return m.stations[i].Username < m.stations[j].Username
		})
		if m.selected >= len(m.stations) && len(m.stations) > 0 {
			m.selected = len(m.stations) - 1
		}

	case errMsg:
		m.err = msg

	case statusMsg:
		m.statusMsg = string(msg)
		cmds = append(cmds, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return statusClearMsg{}
		}))

	case statusClearMsg:
		m.statusMsg = ""

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.pl.Stop()
			return m, tea.Quit

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}

		case "down", "j":
			if m.selected < len(m.stations)-1 {
				m.selected++
			}

		case "p":
			if len(m.stations) > 0 {
				s := m.stations[m.selected]
				if !s.IsLive {
					cmds = append(cmds, func() tea.Msg { return statusMsg("station is offline") })
					break
				}
				if err := m.pl.Start(s.HLSURL); err != nil {
					m.err = fmt.Errorf("could not start ffplay: %w", err)
				} else {
					m.playingURL = s.HLSURL
					cmds = append(cmds, func() tea.Msg { return statusMsg("playing " + s.Username) })
				}
			}

		case "P":
			m.pl.Stop()
			m.playingURL = ""
			cmds = append(cmds, func() tea.Msg { return statusMsg("stopped") })
		}
	}

	return m, tea.Batch(cmds...)
}

func (m listenerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	header := titleStyle.Render("OpenWaves") +
		dimStyle.Render("  "+m.serverURL) +
		dimStyle.Render("  [listener]")
	sb.WriteString(header + "\n")

	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n")
	} else if m.statusMsg != "" {
		sb.WriteString(statusStyle.Render("✓ "+m.statusMsg) + "\n")
	} else {
		sb.WriteString("\n")
	}

	leftWidth := m.width/3 - 2
	if leftWidth < 10 {
		leftWidth = 10
	}
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 10 {
		rightWidth = 10
	}
	innerHeight := m.height - 6
	if innerHeight < 1 {
		innerHeight = 1
	}

	left := m.renderListenerList(leftWidth, innerHeight)
	right := m.renderListenerRight(rightWidth, innerHeight)

	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		boxStyle.Width(leftWidth).Height(innerHeight).Render(left),
		boxStyle.Width(rightWidth).Height(innerHeight).Render(right),
	)
	sb.WriteString(panels + "\n")
	sb.WriteString(m.renderListenerFooter())
	return sb.String()
}

func (m listenerModel) renderListenerList(width, height int) string {
	if len(m.stations) == 0 {
		return dimStyle.Render("No stations")
	}
	var lines []string
	for i, s := range m.stations {
		var badge string
		if s.IsLive {
			badge = liveStyle.Render("● LIVE")
		} else {
			badge = offlineStyle.Render("○ offline")
		}
		line := fmt.Sprintf("%-*s %s", width-10, s.Username, badge)
		if i == m.selected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:min(len(lines), height)], "\n")
}

func (m listenerModel) renderListenerRight(width, height int) string {
	if len(m.stations) == 0 {
		return dimStyle.Render("No stations")
	}
	s := m.stations[m.selected]

	var liveStr string
	if s.IsLive {
		liveStr = liveStyle.Render("● LIVE")
	} else {
		liveStr = offlineStyle.Render("○ offline")
	}

	var playStr string
	if m.pl.IsPlaying() && m.playingURL == s.HLSURL {
		playStr = liveStyle.Render("▶ playing")
	} else {
		playStr = offlineStyle.Render("stopped")
	}

	var detail strings.Builder
	detail.WriteString(titleStyle.Render(s.Name) + "\n")
	detail.WriteString(dimStyle.Render("@"+s.Username) + "\n\n")
	if s.Summary != "" {
		detail.WriteString(truncate(s.Summary, width-2) + "\n\n")
	}
	detail.WriteString(fmt.Sprintf("Status:   %s\n", liveStr))
	detail.WriteString(fmt.Sprintf("Segments: %d\n", s.SegmentCount))
	detail.WriteString(fmt.Sprintf("Player:   %s\n", playStr))
	detail.WriteString("\n")
	detail.WriteString(dimStyle.Render("HLS: "+truncate(s.HLSURL, width-6)) + "\n")
	return detail.String()
}

func (m listenerModel) renderListenerFooter() string {
	hints := "↑/k up  ↓/j down  p: play  P: stop  q: quit"
	return dimStyle.Render(hints)
}

func listenerFetchStations(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		stations, err := client.ListPublicStations()
		if err != nil {
			return errMsg(err)
		}
		return listenerStationsMsg(stations)
	}
}

func listenerTickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}
