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

type Runner struct {
	ScriptPath string
	mu         sync.Mutex
	cmd        *exec.Cmd
	logs       []string
	LogCh      chan string
}

func New(scriptPath string) *Runner {
	return &Runner{
		ScriptPath: scriptPath,
		LogCh:      make(chan string, 20),
	}
}

func (r *Runner) Start(station, serverURL, audioInput string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cmd != nil && r.cmd.Process != nil {
		if err := r.cmd.Process.Signal(syscall.Signal(0)); err == nil {
			return fmt.Errorf("broadcast already running")
		}
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

	r.cmd = cmd
	r.logs = r.logs[:0]

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			r.mu.Lock()
			r.logs = append(r.logs, line)
			if len(r.logs) > maxLogLines {
				r.logs = r.logs[len(r.logs)-maxLogLines:]
			}
			r.mu.Unlock()
			select {
			case r.LogCh <- line:
			default:
			}
		}
	}()

	go func() {
		cmd.Wait()
		pw.Close()
		const ended = "[broadcast ended]"
		r.mu.Lock()
		r.logs = append(r.logs, ended)
		if len(r.logs) > maxLogLines {
			r.logs = r.logs[len(r.logs)-maxLogLines:]
		}
		r.mu.Unlock()
		select {
		case r.LogCh <- ended:
		default:
		}
	}()

	return nil
}

func (r *Runner) Stop() {
	r.mu.Lock()
	cmd := r.cmd
	r.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return
	}

	cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cmd.Process.Signal(syscall.SIGKILL)
	}
}

func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil || r.cmd.Process == nil {
		return false
	}
	return r.cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (r *Runner) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]string, len(r.logs))
	copy(cp, r.logs)
	return cp
}
