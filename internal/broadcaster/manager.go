package broadcaster

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// AudioInputType identifies the audio source for a broadcast.
type AudioInputType string

const (
	AudioSilence  AudioInputType = "silence"   // FFmpeg anullsrc — continuous silence
	AudioTestTone AudioInputType = "test_tone" // FFmpeg sine wave at 440 Hz
	AudioFile     AudioInputType = "file"      // loop a local audio file
)

// AudioInput describes the audio source for a station's broadcast.
type AudioInput struct {
	Type AudioInputType `json:"type"`
	File string         `json:"file,omitempty"` // only for AudioFile
}

// Manager tracks one broadcast subprocess per station.
// It spawns bin/broadcast.sh (or the path in BROADCAST_SCRIPT) on Start and
// kills it on Stop.
type Manager struct {
	mu         sync.Mutex
	processes  map[string]*exec.Cmd
	inputs     map[string]AudioInput
	serverURLs map[string]string
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{
		processes:  make(map[string]*exec.Cmd),
		inputs:     make(map[string]AudioInput),
		serverURLs: make(map[string]string),
	}
}

// SetInput stores an audio input for a station without restarting it.
func (m *Manager) SetInput(username string, input AudioInput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputs[username] = input
}

// GetInput returns the stored audio input for a station, defaulting to AudioSilence.
func (m *Manager) GetInput(username string) AudioInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	if input, ok := m.inputs[username]; ok {
		return input
	}
	return AudioInput{Type: AudioSilence}
}

// audioInputFlag returns the FFmpeg flags string for the given AudioInput.
func audioInputFlag(input AudioInput) string {
	switch input.Type {
	case AudioTestTone:
		return "-f lavfi -i sine=frequency=440"
	case AudioFile:
		return fmt.Sprintf("-stream_loop -1 -i %s", input.File)
	default: // AudioSilence
		return "-f lavfi -i anullsrc=channel_layout=stereo:sample_rate=44100"
	}
}

// startLocked starts the broadcast subprocess. Must be called with m.mu held.
func (m *Manager) startLocked(username, serverURL string) error {
	script := os.Getenv("BROADCAST_SCRIPT")
	if script == "" {
		script = "./bin/broadcast.sh"
	}

	cmd := exec.Command("bash", script, username, serverURL)
	cmd.Env = append(os.Environ(), fmt.Sprintf("AUDIO_INPUT=%s", audioInputFlag(m.inputs[username])))
	// Log subprocess output to the parent process's stderr so logstream captures it.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start broadcast script: %w", err)
	}

	m.processes[username] = cmd
	m.serverURLs[username] = serverURL

	// Reap the process asynchronously so it doesn't become a zombie.
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.processes[username] == cmd {
			delete(m.processes, username)
		}
		m.mu.Unlock()
	}()

	return nil
}

// stopLocked terminates the running subprocess. Must be called with m.mu held.
func (m *Manager) stopLocked(username string) {
	cmd, ok := m.processes[username]
	if !ok {
		return
	}
	delete(m.processes, username)
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		_ = cmd.Process.Kill()
	}
}

// Start spawns the broadcast script for the given station.
// Returns an error if the station is already broadcasting or the script cannot be started.
func (m *Manager) Start(username, serverURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.processes[username]; ok {
		return fmt.Errorf("ingest already running for %s", username)
	}

	m.serverURLs[username] = serverURL
	return m.startLocked(username, serverURL)
}

// Stop terminates the broadcast subprocess for the given station.
// Returns an error if no ingest is running.
func (m *Manager) Stop(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.processes[username]; !ok {
		return fmt.Errorf("no ingest running for %s", username)
	}

	m.stopLocked(username)
	return nil
}

// ChangeInput updates the audio input for a station. If the station is currently
// broadcasting, it is stopped and restarted with the new input.
func (m *Manager) ChangeInput(username string, input AudioInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.inputs[username] = input
	if _, ok := m.processes[username]; !ok {
		return nil // not running, just stored
	}
	serverURL := m.serverURLs[username]
	m.stopLocked(username)
	return m.startLocked(username, serverURL)
}

// IsRunning reports whether a broadcast subprocess is currently active for username.
func (m *Manager) IsRunning(username string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.processes[username]
	return ok
}

