package main

import (
	"path/filepath"
	"strings"
)

type Entry struct {
	ID   int
	Path string
}

func (e Entry) Name() string {
	return filepath.Base(e.Path)
}

func (e Entry) IsEqual(toCompare Entry) bool {
	return EntryIsEqual(e, toCompare)
}

// returns true if e is anywhere in the parent tree of child.
func (e Entry) IsParent(child Entry) bool {
	parentPath := filepath.Clean(e.Path)
	childPath := filepath.Clean(child.Path)

	return strings.HasPrefix(childPath, parentPath+string(filepath.Separator))
}

func EntryIsEqual(a, b Entry) bool {
	return filepath.Clean(a.Path) == filepath.Clean(b.Path)
}

// A [Change] defines a transition. From if the original state of
// [Entry] about to change, and To is the desired state.
//
// when To is nil, we are deleting the [Entry]
type Change struct {
	From Entry
	To   *Entry
}

func (c Change) ID() int {
	return c.From.ID
}
