package main

import "iter"

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{
		m: map[T]struct{}{},
	}

	for _, item := range items {
		s.Insert(item)
	}

	return s
}

func (s *Set[T]) Insert(item T) {
	s.m[item] = struct{}{}
}

func (s *Set[T]) Has(item T) bool {
	_, ok := s.m[item]
	return ok
}

func (s *Set[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s.m {
			if !yield(v) {
				return
			}
		}
	}
}
