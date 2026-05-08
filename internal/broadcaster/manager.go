package broadcaster

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// Manager tracks one broadcast subprocess per station.
// It spawns bin/broadcast.sh (or the path in BROADCAST_SCRIPT) on Start and
// kills it on Stop.
type Manager struct {
	mu        sync.Mutex
	processes map[string]*exec.Cmd
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{processes: make(map[string]*exec.Cmd)}
}

// Start spawns the broadcast script for the given station.
// serverURL is passed as the second positional argument to broadcast.sh.
// audioFile, if non-empty, sets AUDIO_INPUT to "-stream_loop -1 -i <audioFile>".
// Returns an error if the station is already broadcasting or the script cannot be started.
func (m *Manager) Start(username, serverURL, audioFile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.processes[username]; ok {
		return fmt.Errorf("ingest already running for %s", username)
	}

	script := os.Getenv("BROADCAST_SCRIPT")
	if script == "" {
		script = "./bin/broadcast.sh"
	}

	cmd := exec.Command("bash", script, username, serverURL)
	if audioFile != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("AUDIO_INPUT=-stream_loop -1 -i %s", audioFile))
	}
	// Log subprocess output to the parent process's stderr so logstream captures it.
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start broadcast script: %w", err)
	}

	m.processes[username] = cmd

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

// Stop terminates the broadcast subprocess for the given station.
// Returns an error if no ingest is running.
func (m *Manager) Stop(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cmd, ok := m.processes[username]
	if !ok {
		return fmt.Errorf("no ingest running for %s", username)
	}

	delete(m.processes, username)

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		// Fall back to kill if interrupt fails (e.g. process already gone).
		_ = cmd.Process.Kill()
	}
	return nil
}

// IsRunning reports whether a broadcast subprocess is currently active for username.
func (m *Manager) IsRunning(username string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.processes[username]
	return ok
}

