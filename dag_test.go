package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// NOTE: each test must only contain ONE valid output. This is the simplest way
// to to test, and doing so allows us better testing by running each test
// with every perumutation of the input proving that the order in doesn't matter.
func TestDependancyResolution(t *testing.T) {
	tests := []struct {
		name   string
		in     []Change
		expect []Change
	}{
		{
			name:   "single rename",
			in:     createTestChanges("a", "b"),
			expect: createTestChanges("a", "b"),
		},
		{
			name:   "single delete",
			in:     createTestChanges("a", ""),
			expect: createTestChanges("a", ""),
		},
		{
			name: "dependant rename",
			in: createChanges(
				"0;a;b",
				"1;a/a1;x",
			),
			expect: createChanges(
				"1;a/a1;x",
				"0;a;b",
			),
		},
		{
			name: "dependant delete",
			in: createChanges(
				"0;a;b",
				"1;a/a1",
			),
			expect: createChanges(
				"1;a/a1",
				"0;a;b",
			),
		},
		{
			name: "dependant delete and rename",
			in: createChanges(
				"0;a;b",
				"1;a/a1",
				"2;a/a1/a2;y",
			),
			expect: createChanges(
				"2; a/a1/a2; y",
				"1; a/a1",
				"0; a; b",
			),
		},
		{
			name: "rename to a file being deleted",
			in: createChanges(
				"0;a",
				"1;b;a",
			),
			expect: createChanges(
				"1;a",
				"0;b;a",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: implement loop to test all input permutations

			g := NewGraph(tt.in)
			changes, ok := g.OutputChanges()
			if !ok {
				t.Fatalf("output has a circular dependancy")
			}

			require.Equal(t, tt.expect, changes)
		})
	}
}

// test-helper for creating []Change in an easier, more readable way
// string format is: `id;from;to`
//
// Omit the to to signal a delete.
func createChanges(s ...string) []Change {
	changes := make([]Change, len(s))

	for idx, change := range s {
		parts := strings.Split(change, ";")

		var (
			id              int
			idStr, from, to string
			err             error
		)
		switch len(parts) {
		case 2:
			idStr = strings.TrimSpace(parts[0])
			from = strings.TrimSpace(parts[1])
			id, err = strconv.Atoi(idStr)
		case 3:
			idStr = strings.TrimSpace(parts[0])
			from = strings.TrimSpace(parts[1])
			to = strings.TrimSpace(parts[2])
			id, err = strconv.Atoi(idStr)
		default:
			panic("invalid number of parts for a line")
		}
		if err != nil {
			panic("invalid id. must be an integer")
		}

		c := Change{
			From: Entry{id, from},
		}
		if to != "" {
			c.To = &Entry{id, to}
		}
		changes[idx] = c
	}
	return changes
}
