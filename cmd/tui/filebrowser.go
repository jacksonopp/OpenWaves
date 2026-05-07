package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type fsEntry struct {
	name  string
	isDir bool
	path  string
}

type fileBrowser struct {
	dir     string
	entries []fsEntry
	cursor  int
	scroll  int
}

func newFileBrowser() fileBrowser {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	fb := fileBrowser{}
	fb.load(home)
	return fb
}

func (fb *fileBrowser) load(dir string) {
	fb.dir = dir
	fb.cursor = 0
	fb.scroll = 0
	fb.entries = nil

	if dir != filepath.Dir(dir) {
		fb.entries = append(fb.entries, fsEntry{name: "..", isDir: true, path: filepath.Dir(dir)})
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var dirs, mp3s []fsEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		fullPath := filepath.Join(dir, name)
		if e.IsDir() {
			dirs = append(dirs, fsEntry{name: name + "/", isDir: true, path: fullPath})
		} else if strings.ToLower(filepath.Ext(name)) == ".mp3" {
			mp3s = append(mp3s, fsEntry{name: name, isDir: false, path: fullPath})
		}
	}

	sort.Slice(dirs, func(i, j int) bool { return dirs[i].name < dirs[j].name })
	sort.Slice(mp3s, func(i, j int) bool { return mp3s[i].name < mp3s[j].name })

	fb.entries = append(fb.entries, dirs...)
	fb.entries = append(fb.entries, mp3s...)
}

func (fb *fileBrowser) up() {
	if fb.cursor > 0 {
		fb.cursor--
		if fb.cursor < fb.scroll {
			fb.scroll = fb.cursor
		}
	}
}

func (fb *fileBrowser) down(visibleRows int) {
	if fb.cursor < len(fb.entries)-1 {
		fb.cursor++
		if fb.cursor >= fb.scroll+visibleRows {
			fb.scroll = fb.cursor - visibleRows + 1
		}
	}
}

func (fb *fileBrowser) selected() *fsEntry {
	if fb.cursor < 0 || fb.cursor >= len(fb.entries) {
		return nil
	}
	e := fb.entries[fb.cursor]
	return &e
}

func (fb *fileBrowser) visibleEntries(height int) []fsEntry {
	if len(fb.entries) == 0 {
		return nil
	}
	end := fb.scroll + height
	if end > len(fb.entries) {
		end = len(fb.entries)
	}
	return fb.entries[fb.scroll:end]
}
