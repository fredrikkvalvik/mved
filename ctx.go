package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

type FsEntry interface {
	Name() string
	IsDir() bool
}

type MvedContext interface {
	// true if the paths should be printed as absolute paths
	Abs() bool

	// true if the list should be recursive
	Recursive() bool

	// true if deletes are allowed
	Force() bool

	// get the current working dir
	Cwd() string

	FS() afero.Fs

	MatchGlob(p string) bool
	ResolvePath(path string, entry FsEntry) string
	ShouldIgnoreEntry(p string) bool
}

// Ctx hold the main execution context and is responsible
// for handling execution details that should not be direct
// concerns
type Ctx struct {
	config *Config
}

func NewCtx(config *Config) (*Ctx, error) {
	ctx := &Ctx{
		config: config,
	}

	_, err := filepath.Match(ctx.config.Glob, "")
	if err != nil {
		return nil, fmt.Errorf("bad glob pattern: %w", err)
	}

	return ctx, nil
}

var _ MvedContext = (*Ctx)(nil)

func (c *Ctx) Abs() bool {
	return c.config.Abs
}

func (c *Ctx) Recursive() bool {
	return c.config.Recursive
}

func (c *Ctx) Force() bool {
	return c.config.Force
}

func (c *Ctx) Cwd() string {
	return c.config.Cwd
}

func (c *Ctx) MatchGlob(p string) bool {
	if c.config.Glob != "" {
		// we can ignore the error since we did a check at Ctx init.
		ok, _ := filepath.Match(c.config.Glob, p)
		return ok
	}
	return true
}

func (c *Ctx) ShouldIgnoreEntry(name string) bool {
	for _, pattern := range c.config.IgnoredEntries {
		if match, _ := filepath.Match(pattern, name); match {
			return true
		}
	}
	return false
}

func (c *Ctx) ResolvePath(p string, entry FsEntry) (path string) {
	if c.config.Abs && !filepath.IsAbs(p) {
		path = filepath.Clean(filepath.Join(c.config.Cwd, p))
	} else {
		path = strings.TrimPrefix(p, (c.config.Cwd)+string(os.PathSeparator))
	}

	if entry.IsDir() && !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	return path
}

func (c *Ctx) FS() afero.Fs {
	return c.config.Fs
}
