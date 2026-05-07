package player

import (
	"os/exec"
	"sync"
)

// Player manages a single ffplay subprocess for HLS playback.
type Player struct {
	mu  sync.Mutex
	cmd *exec.Cmd
}

func New() *Player { return &Player{} }

// Start launches ffplay for the given HLS URL. Stops any existing playback first.
func (p *Player) Start(hlsURL string) error {
	p.Stop()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", hlsURL)
	return p.cmd.Start()
}

// Stop kills the current ffplay process if running.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		p.cmd = nil
	}
}

// IsPlaying returns true if a playback process is active.
func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// ProcessState is set after Wait(); nil means still running
	return p.cmd.ProcessState == nil
}
