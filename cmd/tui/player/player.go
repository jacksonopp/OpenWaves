package player

import (
	"os/exec"
	"sync"
)

// Player manages a single ffplay subprocess for HLS playback.
type Player struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	isRunning bool
}

func New() *Player { return &Player{} }

// Start launches ffplay for the given HLS URL. Stops any existing playback first.
func (p *Player) Start(hlsURL string) error {
	p.Stop()
	p.mu.Lock()
	defer p.mu.Unlock()
	// -nodisp: no video window; no -autoexit so ffplay keeps polling the live manifest
	p.cmd = exec.Command("ffplay", "-nodisp", "-loglevel", "quiet", hlsURL)
	if err := p.cmd.Start(); err != nil {
		p.cmd = nil
		return err
	}
	p.isRunning = true
	go func() {
		p.cmd.Wait()
		p.mu.Lock()
		p.isRunning = false
		p.mu.Unlock()
	}()
	return nil
}

// Stop kills the current ffplay process if running.
func (p *Player) Stop() {
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	cmd.Wait()

	p.mu.Lock()
	p.isRunning = false
	p.cmd = nil
	p.mu.Unlock()
}

// IsPlaying returns true if a playback process is active.
func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.isRunning
}
