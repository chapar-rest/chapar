package interval

import (
	"fmt"
	"strings"
)

// InvalidIntervalError is a description of an invalid interval.
type InvalidIntervalError string

// Error returns a string representation of the InvalidIntervalError error.
func (s InvalidIntervalError) Error() string {
	return string(s)
}

func newInvalidIntervalError[V, T any](it interval[V, T]) error {
	var b strings.Builder
	fmt.Fprintf(&b, "interval search tree invalid range: start value %v cannot be less than ", it.Start)
	if !it.AllowPoint {
		b.WriteString("or equal to ")
	}
	fmt.Fprintf(&b, "end value %v", it.End)
	return InvalidIntervalError(b.String())
}

// CmpFunc must return a nagative integer, zero or a positive interger as x is
// less than, equal to, or greater than y, respectively.
//
// CmpFunc imposes a total ordering on the given x and y values.
//
// It must also ensure that the relation is transitive: cmp(x, y) > 0 && cmp(y, z) > 0
// implies cmp(x, z) > 0.
type CmpFunc[T any] func(x, y T) int

func (f CmpFunc[T]) eq(x, y T) bool {
	return f(x, y) == 0
}

func (f CmpFunc[T]) lt(x, y T) bool {
	return f(x, y) < 0
}

func (f CmpFunc[T]) lte(x, y T) bool {
	return f(x, y) <= 0
}

func (f CmpFunc[T]) gt(x, y T) bool {
	return f(x, y) > 0
}

func (f CmpFunc[T]) gte(x, y T) bool {
	return f(x, y) >= 0
}

type interval[V, T any] struct {
	Start      T
	End        T
	Val        V
	Vals       []V
	AllowPoint bool
}

func (it interval[V, T]) isInvalid(cmp CmpFunc[T]) bool {
	if it.AllowPoint {
		return cmp.lt(it.End, it.Start)
	}
	return cmp.lte(it.End, it.Start)
}

func (it interval[V, T]) less(start, end T, cmp CmpFunc[T]) bool {
	return cmp.lt(it.Start, start) || cmp.eq(it.Start, start) && cmp.lt(it.End, end)
}

func (it interval[V, T]) intersects(start, end T, cmp CmpFunc[T]) bool {
	return cmp.lte(it.Start, end) && cmp.lte(start, it.End)
}

func (it interval[V, T]) equal(start, end T, cmp CmpFunc[T]) bool {
	return cmp.eq(it.Start, start) && cmp.eq(it.End, end)
}
