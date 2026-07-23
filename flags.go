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
	Recursive bool
	Glob      string
	Path      string
}

func NewFlags(args []string) (Flags, error) {
	f := Flags{
		Force:     false,
		Abs:       false,
		Recursive: false,
		Glob:      "",
		Path:      must(os.Getwd()),
	}

	err := f.Parse(args)
	return f, err
}

func (f *Flags) Parse(args []string) error {
	fs := flag.NewFlagSet("mved", flag.ExitOnError)

	fs.BoolVar(&f.Abs, "a", f.Abs, "Edit the absolute paths instead of relative")
	fs.BoolVar(&f.Force, "f", f.Force, "force flag must be set to delete files")
	fs.BoolVar(&f.Recursive, "r", f.Recursive, "recursively change files from path")
	fs.Var((*globVar)(&f.Glob), "glob", "use a glob pattern to only build a list of files where the file/dir name matches the glob. example: mved -glob \"*.jpeg\"")

	fs.Usage = func() {
		out := fs.Output()
		_, _ = fmt.Fprintln(out, docText())
		_, _ = fmt.Fprintln(out, "Usage:\n\n  mved [flags] [path]\n\nFlags:")

		fs.PrintDefaults()
	}

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	switch fs.NArg() {
	case 0:
	case 1:
		f.Path = fs.Arg(0)
	default:
		return fmt.Errorf("expect 0 or 1 args, got %d", fs.NArg())
	}

	return nil
}

func docText() string {
	return `mved is a tool for renaming/moving/deleting files and direcories using $EDITOR.

mved is able to:
- move files/dirs
- rename files/dirs
- delete files/dirs

All of this is controlled using $EDTIOR.

The way it works is by running the program in a
given directory, the editor opens with a list
of files, where each file has a number at the start of
each line. The number is the ID of a given files and is how
we track changes to a file.

- To move an entry: change the path of the file/dir.
- To rename an entry: change the name of the file/dir.
- To delete an entry: delete the line (or comment out with "#").
`
}

// helper type for adding adding validation to the filepath glob
type globVar string

var _ flag.Value = (*globVar)(nil)

// Set implements [flag.Value].
func (g *globVar) Set(v string) error {
	_, err := filepath.Match(v, "")
	if err != nil {
		return fmt.Errorf("invalid glob: %w", err)
	}
	*g = globVar(v)
	return nil
}

// String implements [flag.Value].
func (g *globVar) String() string {
	return string(*g)
}
