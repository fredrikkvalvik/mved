package main

import "path"

type Entry struct {
	ID    int
	Path  string
	IsDir bool
}

func (e Entry) Name() string {
	return path.Base(e.Path)
}

func EntryIsEqual(a, b Entry) bool {
	return a.Path == b.Path
}

type Change struct {
	From, To Entry
}
