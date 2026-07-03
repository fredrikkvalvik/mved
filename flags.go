package main

import (
	"flag"
	"fmt"
	"os"
)

type Flags struct {
	Force     bool
	Abs       bool
	Help      bool
	Recursive bool
}

func NewFlags() Flags {
	return Flags{
		Force:     false,
		Abs:       false,
		Help:      false,
		Recursive: false,
	}
}

func (f *Flags) Parse() {
	// flag.BoolVar(&flags.Help, "h", flags.Help, "Print help message")
	flag.BoolVar(&f.Abs, "a", f.Abs, "Edit the absolute paths instead of relative")
	flag.BoolVar(&f.Force, "f", f.Force, "force flag must be set to delete files")
	flag.BoolVar(&f.Recursive, "r", f.Recursive, "recursively change files from $CWD")
	flag.Usage = func() {
		_, _ = fmt.Fprint(os.Stdout, help())
	}
	flag.Parse()
}
