package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
			// Changeset only has some ordering guarantees:
			changes, err := BuildChangeset(tt.inParsed, tt.inOriginal)

			require.NoError(t, err)

			require.Equal(t, tt.expect, changes)
		})
	}
}

func TestParentResolution(t *testing.T) {

}
