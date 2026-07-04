package main

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

var fsSimple = fstest.MapFS{
	"a": &fstest.MapFile{},
	"b": &fstest.MapFile{Mode: fs.ModeDir},
}

var fsDeep = fstest.MapFS{
	"a1":       &fstest.MapFile{Mode: fs.ModeDir},
	"a1/b1":    &fstest.MapFile{Mode: fs.ModeDir},
	"a1/b1/c1": &fstest.MapFile{Mode: fs.ModeDir},
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
			filesys: fsSimple,
			expect: []Entry{
				{Path: "a", Name: "a"},
				{Path: "b", Name: "b", IsDir: true},
			},
		},
		{
			name:    "simple recursive",
			flags:   Flags{Recursive: true},
			filesys: fsSimple,
			expect: []Entry{
				{Path: "a", Name: "a"},
				{Path: "b", Name: "b", IsDir: true},
			},
		},
		{
			name:    "nested flat",
			flags:   Flags{Recursive: false},
			filesys: fsDeep,
			expect: []Entry{
				{Path: "a1", Name: "a1", IsDir: true},
			},
		},
		{
			name:    "nested recursive",
			flags:   Flags{Recursive: true},
			filesys: fsDeep,
			expect: []Entry{
				{Path: "a1", Name: "a1", IsDir: true},
				{Path: "a1/b1", Name: "b1", IsDir: true},
				{Path: "a1/b1/c1", Name: "c1", IsDir: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := BuildEntries(tt.flags, tt.filesys)
			require.NoError(t, err)
			t.Logf("parsed: %v", entries)
			require.Equal(t, tt.expect, entries, "lists items must be equal")
		})
	}
}
