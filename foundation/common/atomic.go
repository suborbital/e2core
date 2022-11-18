package common

import "sync/atomic"

// NewAtomicReference returns a new AtomicReference with an initial value.
func NewAtomicReference[T any](init T) *AtomicReference[T] {
	ref := new(AtomicReference[T])
	ref.Store(init)

	return ref
}

// AtomicReference is a Generic wrapper for atomic.Value
type AtomicReference[T any] struct {
	value atomic.Value
}

// Load wraps atomic.Load
func (ref *AtomicReference[T]) Load() T {
	return ref.value.Load().(T)
}

// Store wraps atomic.Store
func (ref *AtomicReference[T]) Store(value T) {
	ref.value.Store(value)
}

// Swap wraps atomic.Swap
func (ref *AtomicReference[T]) Swap(value T) T {
	return ref.value.Swap(value).(T)
}
