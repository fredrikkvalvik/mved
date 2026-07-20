package main

import (
	"fmt"
	"os"
	"path/filepath"

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
	flags          Flags
	cwd            string
	fs             afero.Fs
	ignoredEntries *Set[string]
}

func NewCtx(f Flags, root afero.Fs, ignoredEntries *Set[string]) (*Ctx, error) {
	cwd, _ := os.Getwd()

	ctx := &Ctx{
		flags:          f,
		cwd:            cwd,
		fs:             root,
		ignoredEntries: ignoredEntries,
	}

	_, err := filepath.Match(ctx.flags.Glob, "")
	if err != nil {
		return nil, fmt.Errorf("bad glob pattern: %w", err)
	}

	return ctx, nil
}

var _ MvedContext = (*Ctx)(nil)

func (c *Ctx) Abs() bool {
	return c.flags.Abs
}

func (c *Ctx) Recursive() bool {
	return c.flags.Recursive
}

func (c *Ctx) Force() bool {
	return c.flags.Force
}

func (c *Ctx) Cwd() string {
	return c.cwd
}

func (c *Ctx) MatchGlob(p string) bool {
	if c.flags.Glob != "" {
		// we can ignore the error since we did a check at Ctx init.
		ok, _ := filepath.Match(c.flags.Glob, p)
		return ok
	}
	return true
}

func (c *Ctx) ShouldIgnoreEntry(p string) bool {
	return c.ignoredEntries.Has(p)
}

func (c *Ctx) ResolvePath(p string, entry FsEntry) (path string) {
	if c.flags.Abs {
		path = filepath.Clean(filepath.Join(c.cwd, p))
	} else {
		path = p
	}

	if entry.IsDir() {
		path += string(os.PathSeparator)
	}

	return path
}

func (c *Ctx) FS() afero.Fs {
	return c.fs
}
