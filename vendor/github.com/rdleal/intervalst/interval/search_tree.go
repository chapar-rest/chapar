// Package interval provides a generic interval tree implementation.
//
// An interval tree is a data structure useful for storing values associated with intervals,
// and efficiently search those values based on intervals that overlap with any given interval.
// This generic implementation uses a self-balancing binary search tree algorithm, so searching
// for any intersection has a worst-case time-complexity guarantee of <= 2 log N, where N is the number of items in the tree.
//
// For more on interval trees, see https://en.wikipedia.org/wiki/Interval_tree
//
// To create a tree with time.Time as interval key type and string as value type:
//
//	cmpFn := func(t1, t2 time.Time) int {
//	  switch{
//	  case t1.After(t2): return 1
//	  case t1.Before(t2): return -1
//	  default: return 0
//	  }
//	}
//	st := interval.NewSearchTree[string](cmpFn)
package interval

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"sync"
)

// TreeConfig contains configuration fields that are used to customize the behavior
// of interval trees, specifically SearchTree and MultiValueSearchTree types.
type TreeConfig struct {
	allowIntervalPoint bool
}

// TreeOption is a functional option type used to customize the behavior
// of interval trees, such as the SearchTree and MultiValueSearchTree types.
type TreeOption func(*TreeConfig)

// TreeWithIntervalPoint returns a TreeOption function that configures an interval tree to accept intervals
// in which the start and end key values are the same, effectively representing a point rather than a range in the tree.
func TreeWithIntervalPoint() TreeOption {
	return func(c *TreeConfig) {
		c.allowIntervalPoint = true
	}
}

// TypeMismatchError represents an error that occurs when a type mismatch
// is encountered during the decoding of a tree from its gob representation.
// It indicates that the encoded value does not match the expected type.
type TypeMismatchError struct {
	from, to string
}

// Error returns a string representation of the TypeMismatchError error.
func (e TypeMismatchError) Error() string {
	return fmt.Sprintf("interval: cannot decode type %q into type %q", e.from, e.to)
}

// SearchTree is a generic type representing the Interval Search Tree
// where V is a generic value type, and T is a generic interval key type.
// For more details on how to use these configuration options, see the TreeOption
// function and their usage in the NewSearchTreeWithOptions and NewMultiValueSearchTreeWithOptions functions.
type SearchTree[V, T any] struct {
	mu     sync.RWMutex // used to serialize read and write operations
	root   *node[V, T]
	cmp    CmpFunc[T]
	config TreeConfig
}

// NewSearchTree returns an initialized interval search tree.
// The cmp parameter is used for comparing total order of the interval key type T
// when inserting or looking up an interval in the tree.
// For more details on cmp, see the CmpFunc type.
//
// NewSearchTree will panic if cmp is nil.
func NewSearchTree[V, T any](cmp CmpFunc[T]) *SearchTree[V, T] {
	if cmp == nil {
		panic("NewSearchTree: comparison function cmp cannot be nil")
	}
	return &SearchTree[V, T]{
		cmp: cmp,
	}
}

// NewSearchTreeWithOptions returns an initialized interval search tree with custom configuration options.
// The cmp parameter is used for comparing total order of the interval key type T when inserting or looking up an interval in the tree.
// The opts parameter is an optional list of TreeOptions that customize the behavior of the tree,
// such as allowing point intervals using TreeWithIntervalPoint.
//
// NewSearchTreeWithOptions will panic if cmp is nil.
func NewSearchTreeWithOptions[V, T any](cmp CmpFunc[T], opts ...TreeOption) *SearchTree[V, T] {
	if cmp == nil {
		panic("NewSearchTreeWithOptions: comparison function cmp cannot be nil")
	}

	st := &SearchTree[V, T]{
		cmp: cmp,
	}

	for _, opt := range opts {
		opt(&st.config)
	}

	return st
}

// Height returns the max depth of the tree.
func (st *SearchTree[V, T]) Height() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return int(height(st.root))
}

// Size returns the number of intervals in the tree.
func (st *SearchTree[V, T]) Size() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return size(st.root)
}

// IsEmpty returns true if the tree is empty; otherwise, false.
func (st *SearchTree[V, T]) IsEmpty() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.root == nil
}

// GobEncode encodes the tree (compatible with [encoding/gob]).
func (st *SearchTree[V, T]) GobEncode() ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)

	if err := enc.Encode(st.typeName()); err != nil {
		return nil, err
	}

	if err := enc.Encode(st.config.allowIntervalPoint); err != nil {
		return nil, err
	}

	if st.root != nil {
		if err := enc.Encode(st.root); err != nil {
			return nil, err
		}
	}

	return b.Bytes(), nil
}

// GobDecode decodes the tree (compatible with [encoding/gob]).
func (st *SearchTree[V, T]) GobDecode(data []byte) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	b := bytes.NewBuffer(data)
	enc := gob.NewDecoder(b)

	var typeName string
	wantTypeName := st.typeName()

	if err := enc.Decode(&typeName); err != nil {
		return err
	}

	if typeName != wantTypeName {
		return TypeMismatchError{from: typeName, to: wantTypeName}
	}

	if err := enc.Decode(&st.config.allowIntervalPoint); err != nil {
		return err
	}

	if err := enc.Decode(&st.root); err != nil {
		if err != io.EOF {
			return err
		}

		// An EOF error implies that the root
		// wasn't encoded because it was nil
		st.root = nil
	}

	return nil
}

func (st *SearchTree[V, T]) typeName() string {
	return "SearchTree"
}

// MultiValueSearchTree is a generic type representing the Interval Search Tree
// where V is a generic value type, and T is a generic interval key type.
// MultiValueSearchTree can store multiple values for a given interval key.
type MultiValueSearchTree[V, T any] SearchTree[V, T]

// NewMultiValueSearchTree returns an initialized multi value interval search tree.
// The cmp parameter is used for comparing total order of the interval key type T
// when inserting or looking up an interval in the tree.
// For more details on cmp, see the CmpFunc type.
//
// NewMultiValueSearchTree will panic if cmp is nil.
func NewMultiValueSearchTree[V, T any](cmp CmpFunc[T]) *MultiValueSearchTree[V, T] {
	if cmp == nil {
		panic("NewMultiValueSearchTree: comparison function cmp cannot be nil")
	}
	return &MultiValueSearchTree[V, T]{
		cmp: cmp,
	}
}

// NewSearchTreeWithOptions returns an initialized multi-value interval search tree with custom configuration options.
// The cmp parameter is used for comparing total order of the interval key type T when inserting or looking up an interval in the tree.
// The opts parameter is an optional list of TreeOptions that customize the behavior of the tree,
// such as allowing point intervals using TreeWithIntervalPoint.
//
// NewMultiValueSearchTreeWithOptions will panic if cmp is nil.
func NewMultiValueSearchTreeWithOptions[V, T any](cmp CmpFunc[T], opts ...TreeOption) *MultiValueSearchTree[V, T] {
	if cmp == nil {
		panic("NewMultiValueSearchTreeWithOptions: comparison function cmp cannot be nil")
	}

	st := &MultiValueSearchTree[V, T]{
		cmp: cmp,
	}

	for _, opt := range opts {
		opt(&st.config)
	}

	return st
}

// Height returns the max depth of the tree.
func (st *MultiValueSearchTree[V, T]) Height() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return int(height(st.root))
}

// Size returns the number of intervals in the tree.
func (st *MultiValueSearchTree[V, T]) Size() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return size(st.root)
}

// IsEmpty returns true if the tree is empty; otherwise, false.
func (st *MultiValueSearchTree[V, T]) IsEmpty() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.root == nil
}

// GobEncode encodes the tree (compatible with [encoding/gob]).
func (st *MultiValueSearchTree[V, T]) GobEncode() ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var b bytes.Buffer
	enc := gob.NewEncoder(&b)

	if err := enc.Encode(st.typeName()); err != nil {
		return nil, err
	}

	if err := enc.Encode(st.config.allowIntervalPoint); err != nil {
		return nil, err
	}

	if st.root != nil {
		if err := enc.Encode(st.root); err != nil {
			return nil, err
		}
	}

	return b.Bytes(), nil
}

// GobDecode decodes the tree (compatible with [encoding/gob]).
func (st *MultiValueSearchTree[V, T]) GobDecode(data []byte) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	b := bytes.NewBuffer(data)
	enc := gob.NewDecoder(b)

	var typeName string
	wantTypeName := st.typeName()

	if err := enc.Decode(&typeName); err != nil {
		return err
	}

	if typeName != wantTypeName {
		return TypeMismatchError{from: typeName, to: wantTypeName}
	}

	if err := enc.Decode(&st.config.allowIntervalPoint); err != nil {
		return err
	}

	if err := enc.Decode(&st.root); err != nil {
		if err != io.EOF {
			return err
		}

		// An EOF error implies that the root
		// wasn't encoded because it was nil
		st.root = nil
	}

	return nil
}

func (st *MultiValueSearchTree[V, T]) typeName() string {
	return "MultiValueSearchTree"
}
