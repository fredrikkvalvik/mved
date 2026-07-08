package main

import (
	"bytes"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func fsSimple() *fstest.MapFS {
	return &fstest.MapFS{
		"a": &fstest.MapFile{},
		"b": &fstest.MapFile{Mode: fs.ModeDir},
	}
}

func fsDeep() *fstest.MapFS {
	return &fstest.MapFS{
		"a1":       &fstest.MapFile{Mode: fs.ModeDir},
		"a1/b1":    &fstest.MapFile{Mode: fs.ModeDir},
		"a1/b1/c1": &fstest.MapFile{Mode: fs.ModeDir},
	}
}

func TestBuildEntries(t *testing.T) {
	tests := []struct {
		name    string
		flags   Flags
		filesys fs.FS
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
			entries, err := BuildEntries(tt.flags, tt.filesys)
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

func TestBuildChangeset(t *testing.T) {
	tests := []struct {
		name                 string
		inOriginal, inParsed []Entry
		expect               []Change
	}{
		{
			name:   "empty",
			expect: []Change{},
		},

		{
			name:       "one change",
			inOriginal: createTestEntries("a"),
			inParsed:   createTestEntries("b"),
			expect: []Change{
				{
					Entry{0, "a"},
					&Entry{0, "b"},
				},
			},
		},
		{
			name:       "one delete",
			inOriginal: createTestEntries("a"),
			inParsed:   createTestEntries(""),
			expect: []Change{
				{
					Entry{0, "a"},
					nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes, err := BuildChangeset(tt.inParsed, tt.inOriginal)
			require.NoError(t, err)

			require.Equal(t, tt.expect, changes)
		})
	}
}

func TestPipeline(t *testing.T) {
	tests := []struct {
		name        string
		from        []Entry
		to          string
		expect      []Change
		expectError bool
	}{
		{
			name:   "no-op",
			from:   []Entry{},
			to:     "",
			expect: []Change{},
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
			buf := bytes.NewBufferString(tt.to)
			parsedEntries, err := ParseEntries(buf)
			checkErr(t, tt.expectError, err)

			err = ValidatedParsed(parsedEntries, tt.from)
			checkErr(t, tt.expectError, err)

			changes, err := BuildChangeset(parsedEntries, tt.from)
			checkErr(t, tt.expectError, err)

			require.Equal(t, tt.expect, changes)
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
