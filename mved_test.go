package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func fsSimple() afero.Fs {
	root := afero.NewMemMapFs()
	_ = afero.WriteFile(root, "a", nil, 0)
	_ = afero.WriteFile(root, "b", nil, fs.ModeDir)
	return root
}

func fsDeep() afero.Fs {
	root := afero.NewMemMapFs()
	_ = root.MkdirAll("a1/b1/c1", os.ModeDir)
	return root
}

func TestBuildEntries(t *testing.T) {
	tests := []struct {
		name    string
		flags   Flags
		filesys afero.Fs
		expect  []Entry
	}{
		{
			name:    "simple flat",
			flags:   Flags{Recursive: false},
			filesys: fsSimple(),
			expect:  createTestEntries("a", "b"),
		},
		{
			name:    "simple recursive",
			flags:   Flags{Recursive: true},
			filesys: fsSimple(),
			expect:  createTestEntries("a", "b"),
		},
		{
			name:    "nested flat",
			flags:   Flags{Recursive: false},
			filesys: fsDeep(),
			expect:  createTestEntries("a1"),
		},
		{
			name:    "nested recursive",
			flags:   Flags{Recursive: true},
			filesys: fsDeep(),
			expect:  createTestEntries("a1", "a1/b1", "a1/b1/c1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := BuildEntries(tt.flags, tt.filesys, "")
			require.NoError(t, err)
			require.Equal(t, tt.expect, entries, "lists items must be equal")
		})
	}
}

func TestParseEntries(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expect      []Entry
		expectError bool
	}{
		{
			name:   "empty file",
			input:  "",
			expect: []Entry{},
		},
		{
			name:  "single valid entry",
			input: "0 a",
			expect: []Entry{
				{Path: "a"},
			},
		},
		{
			name:  "two valid entries",
			input: "0 a\n1 b",
			expect: []Entry{
				{Path: "a"},
				{ID: 1, Path: "b"},
			},
		},
		{
			name:        "one invalid entry",
			input:       "f a",
			expectError: true,
		},
		{
			name:        "one invalid entry, one valid",
			input:       "f a\n0 b",
			expectError: true,
		},
		{
			// allow in parsing, we should do a separate validate step
			name:  "out of bound entry id",
			input: "999 a",
			expect: []Entry{
				{ID: 999, Path: "a"},
			},
		},
		{
			name:   "comment only",
			input:  "# comment",
			expect: []Entry{},
		},
		{
			name:   "commented entry",
			input:  "# 0 a",
			expect: []Entry{},
		},
		{
			name:  "entry with comment at end",
			input: "0 a # comment",
			expect: []Entry{
				{Path: "a"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBufferString(tt.input)
			actual, err := ParseEntries(buf)
			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, actual)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expect, actual)
			}
		})
	}
}

func TestPipeline(t *testing.T) {
	tests := []struct {
		name            string
		from            []Entry
		editorInput     string
		expectedRenames int
		expectedDeletes int
		expectError     bool
	}{
		{
			name:        "no-op",
			from:        []Entry{},
			editorInput: "",
		},
		{
			name:            "single rename",
			from:            createTestEntries("a"),
			editorInput:     "0 b",
			expectedRenames: 1,
		},
		{
			name:            "single rename from list",
			from:            createTestEntries("a", "b", "c"),
			editorInput:     "0 a\n1 z\n2 c",
			expectedRenames: 1,
		},
		{
			name: "multiple renames from list",
			from: createTestEntries("a", "b", "c"),
			editorInput: createTestEditorLines(
				"0 x",
				"1 y",
				"2 z"),
			expectedRenames: 3,
		},
		{
			name:            "single delete",
			from:            createTestEntries("a"),
			editorInput:     "",
			expectedDeletes: 1,
		},
		{
			name:            "single delete from list",
			from:            createTestEntries("a", "b", "c"),
			editorInput:     "0 a\n2 c",
			expectedDeletes: 1,
		},
		{
			name:            "multiple deletes from list",
			from:            createTestEntries("a", "b", "c"),
			editorInput:     "",
			expectedDeletes: 3,
		},
		{
			name:            "one rename, one delete. lexical order",
			from:            createTestEntries("a", "b"),
			editorInput:     "1 y",
			expectedRenames: 1,
			expectedDeletes: 1,
		},
		{
			name:            "one rename, one delete. reordered",
			from:            createTestEntries("a", "b"),
			editorInput:     "0 x",
			expectedRenames: 1,
			expectedDeletes: 1,
		},
	}

	checkErr := func(t *testing.T, expectError bool, err error) {
		t.Helper()
		if expectError {
			require.Error(t, err)
			return
		} else {
			require.NoError(t, err)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBufferString(tt.editorInput)

			// test parsing entries
			parsedEntries, err := ParseEntries(buf)
			checkErr(t, tt.expectError, err)

			// test validation
			err = ValidatedParsed(parsedEntries, tt.from, true)
			checkErr(t, tt.expectError, err)

			// test building changeset.
			changes, err := BuildChangeset(parsedEntries, tt.from)
			checkErr(t, tt.expectError, err)

			expectedTotal := tt.expectedDeletes + tt.expectedRenames
			if expectedTotal != len(changes) {
				t.Fatalf("expected a total of %d changes, got %d", expectedTotal, len(changes))
			}

			var (
				renameCount int
				deleteCount int
				failed      bool
			)
			for _, change := range changes {
				if change.To == nil {
					deleteCount += 1
				} else {
					renameCount += 1
				}
			}
			if renameCount != tt.expectedRenames {
				t.Logf("renames expected=%d , got=%d", tt.expectedRenames, renameCount)
				failed = true
			}

			if deleteCount != tt.expectedDeletes {
				t.Logf("deletes expected=%d , got=%d", tt.expectedDeletes, deleteCount)
				failed = true
			}

			if failed {
				t.FailNow()
			}
		})
	}
}

// returns a list of entries with incremented IDs.
//
// empty strings are treated as nil and will be skipped in the list.
func createTestEntries(paths ...string) []Entry {
	entries := make([]Entry, 0)

	for idx, path := range paths {
		if path != "" {
			entries = append(entries, Entry{
				ID:   idx,
				Path: path,
			})
		}
	}

	return entries
}

// Create a set of changes using paired paths.
//
// Every pair is the from->to pair of a change.
//
// Empty strings are treated as nil. If From is empty, then To must be empty as well and the change is ignored.
func createTestChanges(paths ...string) []Change {
	if len(paths)&1 > 0 {
		panic("paths must be of even numbers")
	}
	changes := make([]Change, 0)

	idx := -1
	for path := range slices.Chunk(paths, 2) {
		idx += 1
		from, to := path[0], path[1]
		// ignores the changes while incrementing the ID
		if from == "" {
			if to != "" {
				panic("to must be empty if from is empty")
			}
			continue
		}

		change := Change{From: Entry{idx, from}}

		if to != "" {
			change.To = &Entry{idx, to}
		}
		changes = append(changes, change)
	}

	return changes
}

func createTestEditorLines(lines ...string) string {
	var s strings.Builder
	for _, line := range lines {
		fmt.Fprintln(&s, line)
	}
	s.WriteRune('\n')
	return s.String()
}
