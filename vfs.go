// vfs is a visual file system manager. The idea is to be able to
//   - move files/dirs
//   - rename files/dirs
//   - delete files/dirs
//
// All of this is done using $EDTIOR.
//
// The way it works is by running the program in a
// given directory, the editor opens with a list
// of files, where each file has a number at the start of
// each line. The number is the ID of a given files and is how
// we track changes to a file.
//
//   - To move an entry: change the path of the file/dir.
//   - To rename an entry: change the name of the file/dir.
//   - To delete an entry: delete the line.
//
// Supported flags:
//   - -h: print help text
//   - -a: use absolute paths instead of relative.
//   - -r: recursively list files. default is is only current dir
//   - -f: must be set to be able to delete files
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

func main() {
	flags := NewFlags()
	flags.Parse()

	root := os.DirFS(".")

	if flags.Help {
		_, _ = fmt.Fprint(os.Stdout, help())
		os.Exit(0)
	}

	if err := Run(flags, root, os.Stdin, os.Stdout); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

// 1. read entries
// 2. create tmp file and output [id file]
// 3. parse updated file and validate
// 4. execute changes
func Run(flags Flags, root fs.FS, in io.Reader, out io.Writer) error {
	entries, err := BuildEntries(flags, root)
	if err != nil {
		return err
	}

	var entriesBuffer bytes.Buffer
	_, _ = fmt.Fprintln(&entriesBuffer, printEntries(entries))

	editedEntriesBuffer, err := EditEntries(&entriesBuffer, in, out)
	if err != nil {
		return err
	}

	fmt.Println(editedEntriesBuffer)

	// parsedEntries, err := ParseEntries(f.Name())

	return nil
}

// ORDER IS IMPORTANT.
//
// The index of an item is its ID.
func BuildEntries(flags Flags, root fs.FS) ([]Entry, error) {
	var (
		entries []Entry
		err     error
	)
	if flags.Recursive {
		entries, err = readDirRecursive(root)
	} else {
		entries, err = readDir(root)
	}

	return entries, err
}

// Opens the editor and allows the user to change
// the entries list.
func EditEntries(buf io.Reader, stdin io.Reader, stdout io.Writer) (io.Reader, error) {
	f, _ := os.CreateTemp("", "")
	defer os.Remove(f.Name())
	_, err := io.Copy(f, buf)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(os.ExpandEnv("$EDITOR"), f.Name())
	cmd.Stdin = stdin
	cmd.Stdout = stdout

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(f.Name())
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(b), nil
}

// TODO: add tests
func ParseEntries(buf io.Reader) ([]Entry, error) {
	s := bufio.NewScanner(buf)

	var (
		entries   = []Entry{}
		linecount = -1
	)
	for s.Scan() {
		linecount += 1

		line := s.Text()
		line = strings.TrimSpace(line)

		// just a line comment. ignore
		if strings.HasPrefix(line, "#") {
			continue
		}

		// empty line. continue
		if len(line) == 0 {
			continue
		}

		// discard everything after " #", as that would be a comment
		// TODO: make robust, as this does not handle other types of whitespace
		line = strings.SplitN(line, " #", 2)[0]

		// remove possible empty space after comment split
		line = strings.TrimSpace(line)

		// split into what should be id and path
		fields := strings.Fields(line)

		// we expect two fields, id and path
		if len(fields) != 2 {
			return nil, fmt.Errorf("[%d] expected two fields on line, got=%d", linecount, len(fields))
		}

		idStr, entryPath := fields[0], fields[1]

		// expect first field to be the ID
		// try to parse the first field to an integer
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, fmt.Errorf("[%d] id of entry must be an integer", linecount)
		}

		// validate the path to make sure it is valid
		if !fs.ValidPath(entryPath) {
			return nil, fmt.Errorf("[%d] the entry path is invalid", linecount)
		}
		// normalize the path to pick up different text that should point to the same path
		entryPath = path.Clean(entryPath)

		// we now have valid id and path
		entries = append(entries, Entry{ID: id, Path: entryPath})
	}

	if s.Err() != nil {
		return nil, s.Err()
	}

	return entries, nil
}

// TODO: add tests
func BuildChangeset() []Change {
	return nil
}

// TODO: add tests
func ExecuteChangeset() error {
	return nil
}

func readDir(root fs.FS) ([]Entry, error) {
	e, err := fs.ReadDir(root, ".")
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(e))
	for idx := range e {
		entries[idx] = Entry{
			ID:    idx,
			Path:  e[idx].Name(),
			IsDir: e[idx].IsDir(),
		}
	}
	return entries, nil
}

func readDirRecursive(root fs.FS) ([]Entry, error) {
	var (
		entries   = []Entry{}
		linecount = -1
	)
	err := fs.WalkDir(root, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		linecount += 1
		entries = append(entries, Entry{
			ID:    linecount,
			Path:  path,
			IsDir: d.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func printEntries(e []Entry) string {
	var b strings.Builder

	for idx, entry := range e {
		fmt.Fprintf(&b, "%d %s\n", idx, entry.Path)
	}

	return b.String()
}

func help() string {
	return `vfs is a visual file system manager.

The idea is to be able to:
- move files/dirs
- rename files/dirs
- delete files/dirs

All of this is done using $EDTIOR.

The way it works is by running the program in a
given directory, the editor opens with a list
of files, where each file has a number at the start of
each line. The number is the ID of a given files and is how
we track changes to a file.

- To move an entry: change the path of the file/dir.
- To rename an entry: change the name of the file/dir.
- To delete an entry: delete the line.

Supported flags:
-h: print help text
-a: use absolute paths instead of relative.
-r: recursively list files. default is is only current dir
-f: must be set to be able to delete files
`
}
