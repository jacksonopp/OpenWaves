package main

import (
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jacksonopp/openwaves/cmd/tui/api"
	"github.com/jacksonopp/openwaves/cmd/tui/broadcast"
)

type view int

const (
	viewList        view = iota
	viewDetail      view = iota
	viewInput       view = iota // relay URL and manual audio input
	viewAudioMenu   view = iota // choose audio source type
	viewFileBrowser view = iota // browse for MP3
	viewLoopChoice  view = iota // loop or single playthrough
)

type inputMode int

const (
	inputNone        inputMode = iota
	inputRelaySource inputMode = iota
	inputAudioInput  inputMode = iota
)

type model struct {
	client          *api.Client
	runner          *broadcast.Runner
	stations        []api.StationStatus
	selected        int
	currentView     view
	inputMode       inputMode
	textInput       textinput.Model
	logs            []string
	statusMsg       string
	err             error
	width           int
	height          int
	serverURL       string
	audioMenuCursor int
	fb              fileBrowser
	loopCursor      int
	selectedFile    string
}

// Messages

type tickMsg struct{}
type logLineMsg string
type stationsMsg []api.StationStatus
type errMsg error
type statusClearMsg struct{}
type statusMsg string

// Init

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchStations(m.client), listenLogs(m.runner.LogCh))
}

// Update

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		cmds = append(cmds, fetchStations(m.client), tickCmd())

	case stationsMsg:
		m.stations = []api.StationStatus(msg)
		if m.selected >= len(m.stations) && len(m.stations) > 0 {
			m.selected = len(m.stations) - 1
		}

	case logLineMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 50 {
			m.logs = m.logs[len(m.logs)-50:]
		}
		cmds = append(cmds, listenLogs(m.runner.LogCh))

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
		switch m.currentView {
		case viewInput:
			switch msg.String() {
			case "esc":
				m.currentView = viewDetail
				m.inputMode = inputNone
				m.textInput.Blur()
			case "enter":
				val := m.textInput.Value()
				m.textInput.SetValue("")
				m.textInput.Blur()
				m.currentView = viewDetail
				mode := m.inputMode
				m.inputMode = inputNone
				if len(m.stations) == 0 {
					break
				}
				username := m.stations[m.selected].Username
				switch mode {
				case inputRelaySource:
					cmds = append(cmds, doStartRelay(m.client, username, val))
				case inputAudioInput:
					if err := m.runner.Start(username, m.serverURL, val); err != nil {
						m.err = err
					} else {
						cmds = append(cmds, func() tea.Msg { return statusMsg("broadcast started") })
					}
				}
			default:
				var tiCmd tea.Cmd
				m.textInput, tiCmd = m.textInput.Update(msg)
				cmds = append(cmds, tiCmd)
			}

		case viewAudioMenu:
			switch msg.String() {
			case "up", "k":
				if m.audioMenuCursor > 0 {
					m.audioMenuCursor--
				}
			case "down", "j":
				if m.audioMenuCursor < 2 {
					m.audioMenuCursor++
				}
			case "enter":
				switch m.audioMenuCursor {
				case 0: // use script default
					if len(m.stations) > 0 {
						username := m.stations[m.selected].Username
						if err := m.runner.Start(username, m.serverURL, ""); err != nil {
							m.err = err
						} else {
							cmds = append(cmds, func() tea.Msg { return statusMsg("broadcast started") })
						}
					}
					m.currentView = viewDetail
				case 1: // browse for MP3
					m.fb = newFileBrowser()
					m.currentView = viewFileBrowser
				case 2: // enter manually
					m.currentView = viewInput
					m.inputMode = inputAudioInput
					m.textInput.Placeholder = "e.g. -stream_loop -1 -i /path/to/file.mp3"
					m.textInput.Focus()
				}
			case "esc":
				m.currentView = viewDetail
			}

		case viewFileBrowser:
			innerH := m.height - 8
			if innerH < 1 {
				innerH = 1
			}
			switch msg.String() {
			case "up", "k":
				m.fb.up()
			case "down", "j":
				m.fb.down(innerH)
			case "enter":
				e := m.fb.selected()
				if e == nil {
					break
				}
				if e.isDir {
					m.fb.load(e.path)
				} else {
					m.selectedFile = e.path
					m.loopCursor = 0
					m.currentView = viewLoopChoice
				}
			case "esc":
				m.currentView = viewAudioMenu
			}

		case viewLoopChoice:
			switch msg.String() {
			case "up", "k":
				if m.loopCursor > 0 {
					m.loopCursor--
				}
			case "down", "j":
				if m.loopCursor < 1 {
					m.loopCursor++
				}
			case "enter":
				var audioInput string
				if m.loopCursor == 0 {
					audioInput = "-stream_loop -1 -i " + m.selectedFile
				} else {
					audioInput = "-i " + m.selectedFile
				}
				m.currentView = viewDetail
				if len(m.stations) > 0 {
					username := m.stations[m.selected].Username
					if err := m.runner.Start(username, m.serverURL, audioInput); err != nil {
						m.err = err
					} else {
						cmds = append(cmds, func() tea.Msg { return statusMsg("broadcast started") })
					}
				}
			case "esc":
				m.currentView = viewFileBrowser
			}

		default:
			switch msg.String() {
			case "ctrl+c", "q":
				m.runner.Stop()
				return m, tea.Quit

			case "up", "k":
				if m.currentView == viewList && m.selected > 0 {
					m.selected--
				}

			case "down", "j":
				if m.currentView == viewList && m.selected < len(m.stations)-1 {
					m.selected++
				}

			case "enter":
				if m.currentView == viewList && len(m.stations) > 0 {
					m.currentView = viewDetail
				}

			case "esc":
				if m.currentView == viewDetail {
					m.currentView = viewList
				}

			case "b":
				if m.currentView == viewDetail && len(m.stations) > 0 {
					m.currentView = viewAudioMenu
					m.audioMenuCursor = 0
				}

			case "B":
				if m.currentView == viewDetail {
					m.runner.Stop()
					cmds = append(cmds, func() tea.Msg { return statusMsg("broadcast stopped") })
				}

			case "s":
				if m.currentView == viewDetail && len(m.stations) > 0 {
					username := m.stations[m.selected].Username
					cmds = append(cmds, doStartStream(m.client, username))
				}

			case "S":
				if m.currentView == viewDetail && len(m.stations) > 0 {
					username := m.stations[m.selected].Username
					cmds = append(cmds, doStopStream(m.client, username))
				}

			case "r":
				if m.currentView == viewDetail && len(m.stations) > 0 {
					m.currentView = viewInput
					m.inputMode = inputRelaySource
					m.textInput.Placeholder = "e.g. http://localhost:8080/stations/morning-vibes"
					m.textInput.Focus()
				}

			case "x":
				if m.currentView == viewDetail && len(m.stations) > 0 {
					username := m.stations[m.selected].Username
					cmds = append(cmds, doStopRelay(m.client, username))
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// Commands

func fetchStations(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		stations, err := client.ListStations()
		if err != nil {
			return errMsg(err)
		}
		return stationsMsg(stations)
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func listenLogs(logCh chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-logCh
		if !ok {
			return logLineMsg("[broadcast log channel closed]")
		}
		return logLineMsg(line)
	}
}

func doStopStream(client *api.Client, username string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StopStream(username); err != nil {
			return errMsg(err)
		}
		return statusMsg("stream stopped")
	}
}

func doStartStream(client *api.Client, username string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StartStream(username); err != nil {
			return errMsg(err)
		}
		return statusMsg("stream started")
	}
}

func doStartRelay(client *api.Client, username, sourceURL string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StartRelay(username, sourceURL); err != nil {
			return errMsg(err)
		}
		return statusMsg("relay started")
	}
}

func doStopRelay(client *api.Client, username string) tea.Cmd {
	return func() tea.Msg {
		if err := client.StopRelay(username); err != nil {
			return errMsg(err)
		}
		return statusMsg("relay stopped")
	}
}
