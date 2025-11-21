package sets

import (
	"encoding/json"

	"github.com/go-kratos/kit/containers/maps"
)

// Empty is a zero-size struct to use as the value type in the Set map.
type Empty struct{}

// Set is a generic set of comparable items.
type Set[T comparable] struct {
	m maps.Map[T, Empty]
}

// New creates a Set from the given items.
func New[T comparable](items ...T) *Set[T] {
	set := &Set[T]{}
	set.Insert(items...)
	return set
}

// Insert adds items to the set.
func (s *Set[T]) Insert(items ...T) *Set[T] {
	for _, item := range items {
		s.m.Store(item, Empty{})
	}
	return s
}

// Delete removes items from the set.
func (s *Set[T]) Delete(items ...T) *Set[T] {
	for _, item := range items {
		s.m.Delete(item)
	}
	return s
}

// Clear removes all items from the set.
func (s *Set[T]) Clear() *Set[T] {
	s.m.Clear()
	return s
}

// Has checks if the set contains the given item.
func (s *Set[T]) Has(item T) bool {
	_, contained := s.m.Load(item)
	return contained
}

// HasAll checks if the set contains all the given items.
func (s *Set[T]) HasAll(items ...T) bool {
	for _, item := range items {
		if !s.Has(item) {
			return false
		}
	}
	return true
}

// HasAny checks if the set contains any of the given items.
func (s *Set[T]) HasAny(items ...T) bool {
	for _, item := range items {
		if s.Has(item) {
			return true
		}
	}
	return false
}

// ToSlice returns the items in the set as a slice.
func (s *Set[T]) ToSlice() []T {
	items := make([]T, 0)
	s.m.Range(func(item T, _ Empty) bool {
		items = append(items, item)
		return true
	})
	return items
}

// Clone creates a copy of the set.
func (s *Set[T]) Clone() *Set[T] {
	set := New[T]()
	s.m.Range(func(item T, _ Empty) bool {
		set.Insert(item)
		return true
	})
	return set
}

// Len returns the number of items in the set.
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	items := make([]T, 0)
	s.m.Range(func(item T, _ Empty) bool {
		items = append(items, item)
		return true
	})
	return json.Marshal(items)
}

// UnmarshalJSON unmarshals a JSON array into the set.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	var items []T
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	s.Clear()
	s.Insert(items...)
	return nil
}
