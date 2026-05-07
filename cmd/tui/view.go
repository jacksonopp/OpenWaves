package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	liveStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	offlineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
)

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Header
	header := titleStyle.Render("OpenWaves") + dimStyle.Render("  server: "+m.serverURL)
	sb.WriteString(header + "\n")

	// Status/error bar
	if m.err != nil {
		sb.WriteString(errorStyle.Render("Error: "+m.err.Error()) + "\n")
	} else if m.statusMsg != "" {
		sb.WriteString(statusStyle.Render("✓ "+m.statusMsg) + "\n")
	} else {
		sb.WriteString("\n")
	}

	// Two-panel layout
	leftWidth := m.width/3 - 2
	if leftWidth < 10 {
		leftWidth = 10
	}
	rightWidth := m.width - leftWidth - 4
	if rightWidth < 10 {
		rightWidth = 10
	}

	innerHeight := m.height - 6 // header + statusbar + footer + borders
	if innerHeight < 1 {
		innerHeight = 1
	}

	left := m.renderList(leftWidth, innerHeight)
	right := m.renderRight(rightWidth, innerHeight)

	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		boxStyle.Width(leftWidth).Height(innerHeight).Render(left),
		boxStyle.Width(rightWidth).Height(innerHeight).Render(right),
	)
	sb.WriteString(panels + "\n")

	// Footer
	sb.WriteString(m.renderFooter())

	return sb.String()
}

func (m model) renderList(width, height int) string {
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
		if i == m.selected && m.currentView != viewList {
			line = selectedStyle.Render(line)
		} else if i == m.selected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	// Pad to height
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines[:min(len(lines), height)], "\n")
}

func (m model) renderRight(width, height int) string {
	switch m.currentView {
	case viewInput:
		return m.renderInputPanel(width)
	case viewAudioMenu:
		return m.renderAudioMenu()
	case viewFileBrowser:
		return m.renderFileBrowser(width, height)
	case viewLoopChoice:
		return m.renderLoopChoice()
	default:
		if m.currentView == viewDetail && len(m.stations) > 0 {
			return m.renderDetail(width, height)
		}
		return dimStyle.Render("Select a station and press enter")
	}
}

func (m model) renderInputPanel(width int) string {
	var prompt string
	switch m.inputMode {
	case inputRelaySource:
		prompt = "Start relay — enter source URL:"
	case inputAudioInput:
		prompt = "Start broadcast — enter audio input:"
	default:
		prompt = "Input:"
	}
	return fmt.Sprintf("%s\n\n%s", titleStyle.Render(prompt), m.textInput.View())
}

func (m model) renderDetail(width, height int) string {
	s := m.stations[m.selected]

	var liveStr string
	if s.IsLive {
		liveStr = liveStyle.Render("● LIVE")
	} else {
		liveStr = offlineStyle.Render("○ offline")
	}

	var relayStr string
	if s.IsRelaying {
		relayStr = liveStyle.Render("relaying")
	} else {
		relayStr = offlineStyle.Render("not relaying")
	}

	var broadcastStr string
	if m.runner.IsRunning() {
		broadcastStr = liveStyle.Render("running")
	} else {
		broadcastStr = offlineStyle.Render("stopped")
	}

	var detail strings.Builder
	detail.WriteString(titleStyle.Render(s.Username) + "\n")
	detail.WriteString(fmt.Sprintf("Status:    %s\n", liveStr))
	detail.WriteString(fmt.Sprintf("Segments:  %d\n", s.SegmentCount))
	detail.WriteString(fmt.Sprintf("Relay:     %s\n", relayStr))
	detail.WriteString(fmt.Sprintf("Broadcast: %s\n", broadcastStr))
	detail.WriteString("\n")
	detail.WriteString(dimStyle.Render("─── broadcast log ───") + "\n")

	logs := m.logs
	if len(logs) > 8 {
		logs = logs[len(logs)-8:]
	}
	for _, line := range logs {
		detail.WriteString(dimStyle.Render(truncate(line, width-2)) + "\n")
	}

	return detail.String()
}

func (m model) renderFooter() string {
	var hints string
	switch m.currentView {
	case viewList:
		hints = "↑/k up  ↓/j down  enter: detail  q: quit"
	case viewDetail:
		hints = "esc: back  b: start broadcast  B: stop broadcast  s: start stream  S: stop stream  r: relay  x: stop relay  q: quit"
	case viewInput:
		hints = "enter: confirm  esc: cancel"
	case viewAudioMenu:
		hints = "↑/k up  ↓/j down  enter: select  esc: cancel"
	case viewFileBrowser:
		hints = "↑/k up  ↓/j down  enter: open/select  esc: back"
	case viewLoopChoice:
		hints = "↑/k up  ↓/j down  enter: confirm  esc: back"
	}
	return dimStyle.Render(hints)
}

func (m model) renderAudioMenu() string {
	options := []string{
		"Use script default (test tone)",
		"Browse for MP3 file",
		"Enter FFmpeg flags manually",
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Start Broadcast") + "\n\n")
	for i, opt := range options {
		if i == m.audioMenuCursor {
			sb.WriteString(selectedStyle.Render("► "+opt) + "\n")
		} else {
			sb.WriteString("  " + opt + "\n")
		}
	}
	return sb.String()
}

func (m model) renderFileBrowser(width, height int) string {
	innerH := height - 3
	if innerH < 1 {
		innerH = 1
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Browse for MP3") + "\n")
	sb.WriteString(dimStyle.Render(truncate(m.fb.dir, width-2)) + "\n")

	if len(m.fb.entries) == 0 {
		sb.WriteString(dimStyle.Render("(empty)") + "\n")
		return sb.String()
	}

	visible := m.fb.visibleEntries(innerH)
	for i, e := range visible {
		absIdx := i + m.fb.scroll
		var line string
		if e.isDir {
			line = "[DIR] " + e.name
		} else {
			line = "[MP3] " + e.name
		}
		line = truncate(line, width-4)
		if absIdx == m.fb.cursor {
			line = selectedStyle.Render("► " + line)
		} else {
			line = "  " + line
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func (m model) renderLoopChoice() string {
	options := []string{
		"Loop indefinitely",
		"Play once",
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Playback Mode") + "\n")
	sb.WriteString(dimStyle.Render(truncate(m.selectedFile, 40)) + "\n\n")
	for i, opt := range options {
		if i == m.loopCursor {
			sb.WriteString(selectedStyle.Render("► "+opt) + "\n")
		} else {
			sb.WriteString("  " + opt + "\n")
		}
	}
	return sb.String()
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
