package broadcast

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const maxLogLines = 50

// stationProc tracks a single running broadcast process for one station.
type stationProc struct {
	cmd  *exec.Cmd
	done chan struct{} // closed when the process exits
	mu   sync.Mutex
	logs []string
}

func (p *stationProc) addLog(line string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.logs = append(p.logs, line)
	if len(p.logs) > maxLogLines {
		p.logs = p.logs[len(p.logs)-maxLogLines:]
	}
}

func (p *stationProc) getLines() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make([]string, len(p.logs))
	copy(cp, p.logs)
	return cp
}

func (p *stationProc) isRunning() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

// Runner manages per-station broadcast processes.
type Runner struct {
	ScriptPath string
	mu         sync.Mutex
	procs      map[string]*stationProc
}

func New(scriptPath string) *Runner {
	return &Runner{
		ScriptPath: scriptPath,
		procs:      make(map[string]*stationProc),
	}
}

// Start launches a broadcast for the given station. Returns an error if a
// broadcast is already running for that station.
func (r *Runner) Start(station, serverURL, audioInput string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if proc, ok := r.procs[station]; ok && proc.isRunning() {
		return fmt.Errorf("broadcast already running for %s", station)
	}

	cmd := exec.Command(r.ScriptPath, station, serverURL)
	if audioInput != "" {
		cmd.Env = append(cmd.Environ(), "AUDIO_INPUT="+audioInput)
	}

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return err
	}

	proc := &stationProc{
		cmd:  cmd,
		done: make(chan struct{}),
	}
	r.procs[station] = proc

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			proc.addLog(scanner.Text())
		}
	}()

	go func() {
		cmd.Wait()
		pw.Close()
		proc.addLog("[broadcast ended]")
		close(proc.done)
	}()

	return nil
}

// Stop terminates the broadcast for the given station.
func (r *Runner) Stop(station string) {
	r.mu.Lock()
	proc := r.procs[station]
	r.mu.Unlock()

	if proc == nil || !proc.isRunning() {
		return
	}

	proc.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-proc.done:
	case <-time.After(2 * time.Second):
		proc.cmd.Process.Signal(syscall.SIGKILL)
		<-proc.done
	}
}

// StopAll terminates all running broadcasts (used on quit).
func (r *Runner) StopAll() {
	r.mu.Lock()
	stations := make([]string, 0, len(r.procs))
	for s := range r.procs {
		stations = append(stations, s)
	}
	r.mu.Unlock()

	for _, s := range stations {
		r.Stop(s)
	}
}

// IsRunning reports whether a broadcast is active for the given station.
func (r *Runner) IsRunning(station string) bool {
	r.mu.Lock()
	proc := r.procs[station]
	r.mu.Unlock()
	return proc != nil && proc.isRunning()
}

// Lines returns the most recent log lines for the given station's broadcast.
func (r *Runner) Lines(station string) []string {
	r.mu.Lock()
	proc := r.procs[station]
	r.mu.Unlock()
	if proc == nil {
		return nil
	}
	return proc.getLines()
}
