package utils

type Set[T comparable] struct {
	items map[T]bool
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		items: make(map[T]bool),
	}
}

func (s *Set[T]) Add(item T) {
	s.items[item] = true
}

func (s *Set[T]) Contains(item T) bool {
	_, exists := s.items[item]
	return exists
}

func (s *Set[T]) Size() int {
	return len(s.items)
}

func (s *Set[T]) Items() []T {
	keys := make([]T, 0, len(s.items))
	for item := range s.items {
		keys = append(keys, item)
	}
	return keys
}
