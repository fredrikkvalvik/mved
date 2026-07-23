package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

type Config struct {
	Force     bool
	Abs       bool
	Recursive bool
	Glob      string
	Cwd       string

	Fs afero.Fs

	IgnoredEntries []string
}

// fs defaults to [afero.OsFs]
func ResolveConfig(f Flags, opts ...configOpt) (*Config, error) {
	c := Config{
		Force:          f.Force,
		Abs:            f.Abs,
		Recursive:      f.Recursive,
		Glob:           f.Glob,
		IgnoredEntries: f.Ignores,
	}
	for _, opt := range opts {
		opt(&c)
	}

	// we only resolve the path when Cwd is empty
	if c.Cwd == "" {
		var err error
		cwd := f.Path
		if cwd == "" {
			c.Cwd, err = os.Getwd()
		} else {
			c.Cwd, err = filepath.Abs(cwd)
		}

		if err != nil {
			return nil, err
		}
	}

	// default to Os fs
	if c.Fs == nil {
		c.Fs = afero.NewOsFs()
	}

	// default to non-nil empty set
	if c.IgnoredEntries == nil {
		c.IgnoredEntries = []string{}
	}

	return &c, nil
}

type configOpt func(c *Config)

// add ignored entries to config
func WithIgnores(ignoredEntries ...string) configOpt {
	return func(c *Config) {
		c.IgnoredEntries = append(c.IgnoredEntries, ignoredEntries...)
	}
}

// overwrite the default filesystem
func WithFs(root afero.Fs) configOpt {
	return func(c *Config) {
		c.Fs = root
	}
}

// overwrite the os CWD (for testing)
func WithCwd(cwd string) configOpt {
	return func(c *Config) {
		c.Cwd = cwd
	}
}
