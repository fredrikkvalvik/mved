package main

type Entry struct {
	Path  string
	Name  string
	IsDir bool
}

func EntryIsEqual(a, b Entry) bool {
	return a.Name == b.Name &&
		a.Path == b.Path
}

type Change struct {
	From, To Entry
}
