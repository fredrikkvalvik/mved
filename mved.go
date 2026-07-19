// mved is a tool for renaming/moving/deleting files and direcories using $EDITOR.
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
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/afero"
)

func main() {
	flags := NewFlags()
	root := afero.NewOsFs()

	if flags.Help {
		_, _ = fmt.Fprint(os.Stdout, help())
		os.Exit(0)
	}

	ctx, err := NewCtx(flags, root, NewSet(
		".git",
		"node_modules",
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Run(ctx MvedContext) error {
	entries, err := BuildEntries(ctx)
	if err != nil {
		return err
	}

	var entriesBuffer bytes.Buffer
	fmt.Fprintln(&entriesBuffer, "# edit a line to rename an entry")
	fmt.Fprintln(&entriesBuffer, "# remove/comment out a line to delete an entry")
	fmt.Fprintln(&entriesBuffer, "# the number at the start is the file ID. It is used to detmermine what to do with the entry")
	fmt.Fprintln(&entriesBuffer, "")

	_, _ = fmt.Fprintln(&entriesBuffer, printEntries(entries))

	editedEntriesBuffer, err := editEntries(&entriesBuffer)
	if err != nil {
		return err
	}

	parsed, err := ParseEntries(editedEntriesBuffer)
	if err != nil {
		return err
	}

	err = ValidatedParsed(parsed, entries, ctx.Force())
	if err != nil {
		return err
	}

	changes, err := BuildChangeset(parsed, entries)
	if err != nil {
		return err
	}

	graph := NewGraph(changes)
	changes, ok := graph.OutputChanges()
	if !ok {
		return fmt.Errorf("graph is not acyclic")
	}

	err = ExecuteChangeset(ctx.FS(), changes)
	if err != nil {
		return err
	}

	return nil
}

// ORDER IS IMPORTANT.
//
// The index of an item is its ID.
//
// TODO: implement logic for pointing to another dir than "." for listing/writing
func BuildEntries(ctx MvedContext) ([]Entry, error) {
	var (
		entries []Entry
		err     error
	)
	if ctx.Recursive() {
		entries, err = readDirRecursive(ctx)
	} else {
		entries, err = readDir(ctx)
	}

	return entries, err
}

func isIgnoredEntry(ignored *Set[string], entry string) bool {
	return ignored.Has(entry)
}

// Opens the editor and allows the user to change
// the entries list.
func editEntries(buf io.Reader) (io.Reader, error) {
	f, _ := os.CreateTemp("", "")
	defer func() {
		_ = os.Remove(f.Name())
	}()

	_, err := io.Copy(f, buf)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(os.ExpandEnv("$EDITOR"), f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

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

		// normalize the path to pick up different text that should point to the same path
		entryPath = filepath.Clean(entryPath)

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
func ValidatedParsed(parsed, original []Entry, allowDeletes bool) error {
	errs := []error{}

	occurences := map[int]struct{}{}
	duplicates := map[string]int{}

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

		if first, found := duplicates[entry.Path]; !found {
			duplicates[entry.Path] = entry.ID
		} else {
			errs = append(errs, fmt.Errorf("[%d] duplicate path: \"%s\" of [%d]", entry.ID, entry.Path, first))
		}
	}

	if !allowDeletes {
		for id := range original {
			_, found := occurences[id]
			if !found {
				errs = append(errs, fmt.Errorf("[%d] delete without force flag is not allowed", id))
			}
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
func BuildChangeset(parsed, original []Entry) ([]Change, error) {
	var (
		// all entries are "marked for deletion" by default.
		// we need to see the entry to "unmark" it when
		// looping the parsed entries
		occurrences = map[int]struct{}{}

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

		changes = append(changes, Change{
			From: from,
			To:   &to,
		})
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
	return changes, nil
}

// NOTE: [fs.FS] does not define any remove methods,
// meaning that we aren't able to pass an [fs.FS] to test this.
func ExecuteChangeset(root afero.Fs, changes []Change) error {
	for _, change := range changes {
		changeIsDelete := change.To == nil

		if changeIsDelete {
			// remove
			err := root.RemoveAll(change.From.Path)
			if err != nil {
				return err
			}
		} else {
			err := ensureParentDirExists(change.To.Path, root)
			if err != nil {
				return err
			}
			// move/rename
			err = root.Rename(change.From.Path, change.To.Path)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureParentDirExists(target string, root afero.Fs) error {
	targetParentPath := path.Dir(target)
	parentExists, err := afero.DirExists(root, targetParentPath)
	if err != nil {
		return err
	}
	if !parentExists {
		err := root.MkdirAll(targetParentPath, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func readDir(ctx MvedContext) ([]Entry, error) {
	e, err := afero.ReadDir(ctx.FS(), ".")
	if err != nil {
		return nil, err
	}

	// Logic flows as follows:
	//	1. create a list of entries.
	//	2. check to see if the current file matches glob.
	//	3. on match, append to entries slice using the pre-append length of entries as ID

	var entries []Entry
	for _, entry := range e {
		// skip ignored entries
		if ctx.ShouldIgnoreEntry(entry.Name()) {
			continue
		}

		if ok := ctx.MatchGlob(entry.Name()); ok {
			entries = append(entries, Entry{
				ID:   len(entries),
				Path: ctx.ResolvePath(entry.Name()),
			})
		}
	}

	return entries, nil
}

func readDirRecursive(ctx MvedContext) ([]Entry, error) {
	var (
		entries   = []Entry{}
		linecount = -1
	)

	err := afero.Walk(ctx.FS(), ".", func(path string, info fs.FileInfo, err error) error {
		if path == "." {
			return nil
		}

		// skip ignored entries
		if ctx.ShouldIgnoreEntry(info.Name()) {
			return fs.SkipDir
		}

		if match := ctx.MatchGlob(path); match {
			linecount += 1
			entries = append(entries, Entry{
				ID:   linecount,
				Path: ctx.ResolvePath(path),
			})
		}
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

Supported flags:
-h: print help text
-a: use absolute paths instead of relative.
-r: recursively list files. default is is only current dir
-f: must be set to be able to delete files
`
}
