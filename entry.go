package main

import "io/fs"

type Entry struct {
	Path  string
	Name  string
	IsDir bool
	Ref   fs.DirEntry
}
