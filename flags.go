package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Flags struct {
	Force     bool
	Abs       bool
	Help      bool
	Recursive bool
	Glob      string
	Cwd       string
}

func NewFlags() Flags {
	f := Flags{
		Force:     false,
		Abs:       false,
		Help:      false,
		Recursive: false,
		Glob:      "",
		Cwd:       must(os.Getwd()),
	}

	f.Parse()
	return f
}

func (f *Flags) Parse() {
	// flag.BoolVar(&flags.Help, "h", flags.Help, "Print help message")
	flag.BoolVar(&f.Abs, "a", f.Abs, "Edit the absolute paths instead of relative")
	flag.BoolVar(&f.Force, "f", f.Force, "force flag must be set to delete files")
	flag.BoolVar(&f.Recursive, "r", f.Recursive, "recursively change files from path")
	flag.StringVar(&f.Glob, "glob", f.Glob, "use a glob pattern to only build a list of files where the file/dir name matches the glob. example: mved -glob \"*.jpeg\"")
	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stdout, help())
	}

	flag.Parse()
	if cwd := flag.Arg(0); cwd != "" {
		f.Cwd = must(filepath.Abs(cwd))

	}
}

func must[T any](v T, err error) T {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	return v
}
