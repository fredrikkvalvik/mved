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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"unicode"
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
		fmt.Fprintf(os.Stderr, "%v\n", err)
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

	editedEntriesBuffer, err := editEntries(&entriesBuffer, in, out)
	if err != nil {
		return err
	}

	parsed, err := ParseEntries(editedEntriesBuffer)
	if err != nil {
		return err
	}

	err = ValidatedParsed(parsed, entries)
	if err != nil {
		return err
	}

	changes, err := BuildChangeset(parsed, entries)
	if err != nil {
		return err
	}

	fmt.Println(changes)

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
func editEntries(buf io.Reader, stdin io.Reader, stdout io.Writer) (io.Reader, error) {
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

// Parse the a buffer, line by line, and
// try to parse each line into an [Entry].
func ParseEntries(buf io.Reader) ([]Entry, error) {
	s := bufio.NewScanner(buf)

	var (
		entries   = []Entry{}
		linecount = -1
		errs      []error
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

		// scan  the line and look for a comment.
		commentIndex := indexOfLineComment(line)

		// comment found, strip away
		if commentIndex > -1 {
			line = string([]rune(line)[0:commentIndex])
		}

		// remove possible empty space after comment split
		line = strings.TrimSpace(line)

		// split into what should be id and path
		fields := strings.Fields(line)

		// we expect two fields, id and path
		if len(fields) != 2 {
			errs = append(errs, fmt.Errorf("[%d] expected two fields on line, got=%d", linecount, len(fields)))
			continue
		}

		idStr, entryPath := fields[0], fields[1]

		// expect first field to be the ID
		// try to parse the first field to an integer
		id, err := strconv.Atoi(idStr)
		if err != nil {
			errs = append(errs, fmt.Errorf("[%d] id of entry must be an integer", linecount))
			continue
		}

		// validate the path to make sure it is valid
		if !fs.ValidPath(entryPath) {
			errs = append(errs, fmt.Errorf("[%d] the entry path is invalid", linecount))
			continue
		}
		// normalize the path to pick up different text that should point to the same path
		entryPath = path.Clean(entryPath)

		// we now have valid id and path
		entries = append(entries, Entry{ID: id, Path: entryPath})
	}

	if s.Err() != nil {
		return nil, s.Err()
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return entries, nil
}

// return index of the rune where a space is followed by a '#'
//
// if noe comment is found, return -1
func indexOfLineComment(line string) int {
	// start out true to handle the line starting with '#'
	prevSpace := true
	for idx, ch := range line {
		if unicode.IsSpace(ch) {
			prevSpace = true
			continue
		}
		if ch == '#' && prevSpace {
			return idx
		}

		// reset prevSpace as none of the above was true
		prevSpace = false
	}
	return -1
}

// run validations on the parsed list of entries
func ValidatedParsed(parsed, original []Entry) error {
	errs := []error{}

	occurences := map[int]struct{}{}

	for _, entry := range parsed {
		// make sure the ID is somewhere within the allowed boundary
		if entry.ID >= len(original) || entry.ID < 0 {
			errs = append(errs, fmt.Errorf("[%d] invalid id", entry.ID))
		}

		// make sure that no ID occurs more than once
		if _, found := occurences[entry.ID]; !found {
			occurences[entry.ID] = struct{}{}
		} else {
			errs = append(errs, fmt.Errorf("[%d] multiple occurences. only one is allowed", entry.ID))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// BuildChangeset generates a list of changes by first
// comparing the lists and see what actual changes have been made,
// and then creating a DAG to resolve the order renames occur.
//
// deletions are done before moves.
func BuildChangeset(parsed, original []Entry) ([]Change, error) {
	var (
		// all entries are "marked for deletion" by default.
		// we need to see the entry to "unmark" it when
		// looping the parsed entries
		occurrences   = map[int]struct{}{}
		renameChanges = map[int]Change{}
		renameNodes   = []Entry{}

		changes = []Change{}
	)

	for _, to := range parsed {
		// add to occurences map since the item still existed in parsed entries
		occurrences[to.ID] = struct{}{}

		from := original[to.ID]

		// if entries are equal, no change is needed
		if from.IsEqual(to) {
			continue
		}

		renameNodes = append(renameNodes, from)
		renameChanges[from.ID] = Change{
			From: from,
			To:   &to,
		}
	}

	// iterate original list and look for deletions
	for idx, e := range original {
		_, occured := occurrences[idx]
		if !occured {
			changes = append(changes, Change{
				From: e,
			})
		}
	}

	// build the DAG for renames
	graph := NewGraph(renameNodes)
	graph.ComputeEdges()
	entriesOrdered, ok := graph.OutputChanges()
	if !ok {
		return nil, fmt.Errorf("circular dependancy")
	}
	for _, e := range entriesOrdered {
		changes = append(changes, renameChanges[e.ID])
	}

	return changes, nil
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
			ID:   idx,
			Path: e[idx].Name(),
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
			ID:   linecount,
			Path: path,
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
