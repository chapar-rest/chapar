package interval

// Find returns the value which interval key exactly matches with the given start and end interval.
// It returns true as the second return value if an exaclty matching interval key is found in the tree;
// otherwise, false.
func (st *SearchTree[V, T]) Find(start, end T) (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V

	interval, ok := find(st.root, start, end, st.cmp)
	if !ok {
		return val, false
	}

	return interval.Val, true
}

func find[V, T any](root *node[V, T], start, end T, cmp CmpFunc[T]) (interval[V, T], bool) {
	if root == nil {
		return interval[V, T]{}, false
	}

	cur := root
	for cur != nil {
		switch {
		case cur.Interval.equal(start, end, cmp):
			return cur.Interval, true
		case cur.Interval.less(start, end, cmp):
			cur = cur.Right
		default:
			cur = cur.Left
		}
	}

	return interval[V, T]{}, false
}

// AnyIntersection returns a value which interval key intersects with the given start and end interval.
// It returns true as the second return value if any intersection is found in the tree; otherwise, false.
func (st *SearchTree[V, T]) AnyIntersection(start, end T) (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V

	interval, ok := anyIntersections(st.root, start, end, st.cmp)
	if !ok {
		return val, false
	}

	return interval.Val, true
}

func anyIntersections[V, T any](root *node[V, T], start, end T, cmp CmpFunc[T]) (interval[V, T], bool) {
	if root == nil {
		return interval[V, T]{}, false
	}

	cur := root
	for cur != nil {
		if cur.Interval.intersects(start, end, cmp) {
			return cur.Interval, true
		}

		next := cur.Left
		if cur.Left == nil || cmp.gt(start, cur.Left.MaxEnd) {
			next = cur.Right
		}

		cur = next
	}

	return interval[V, T]{}, false
}

// AllIntersections returns a slice of values which interval key intersects with the given start and end interval.
// It returns true as the second return value if any intersection is found in the tree; otherwise, false.
func (st *SearchTree[V, T]) AllIntersections(start, end T) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V
	if st.root == nil {
		return vals, false
	}

	searchInOrder(st.root, start, end, st.cmp, func(it interval[V, T]) {
		vals = append(vals, it.Val)
	})

	return vals, len(vals) > 0
}

func searchInOrder[V, T any](n *node[V, T], start, end T, cmp CmpFunc[T], foundFn func(interval[V, T])) {
	if n.Left != nil && cmp.lte(start, n.Left.MaxEnd) {
		searchInOrder(n.Left, start, end, cmp, foundFn)
	}

	if n.Interval.intersects(start, end, cmp) {
		foundFn(n.Interval)
	}

	if n.Right != nil && cmp.lte(n.Interval.Start, end) {
		searchInOrder(n.Right, start, end, cmp, foundFn)
	}
}

// Min returns the value which interval key is the minimum interval key in the tree.
// It returns false as the second return value if the tree is empty; otherwise, true.
func (st *SearchTree[V, T]) Min() (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V
	if st.root == nil {
		return val, false
	}

	val = min(st.root).Interval.Val

	return val, true
}

// Max returns the value which interval key is the maximum interval in the tree.
// It returns false as the second return value if the tree is empty; otherwise, true.
func (st *SearchTree[V, T]) Max() (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V
	if st.root == nil {
		return val, false
	}

	val = max(st.root).Interval.Val

	return val, true
}

// MaxEnd returns the values in the tree that have the largest ending interval.
// It returns false as the second return value if the tree is empty; otherwise, true.
func (st *SearchTree[V, T]) MaxEnd() ([]V, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	var vals []V
	if st.root == nil {
		return vals, false
	}

	maxEnd(st.root, st.root.MaxEnd, st.cmp, func(n *node[V, T]) {
		vals = append(vals, n.Interval.Val)
	})
	return vals, true
}

// Ceil returns a value which interval key is the smallest interval key greater than the given start and end interval.
// It returns true as the second return value if there's a ceiling interval key for the given start and end interval
// in the tree; otherwise, false.
func (st *SearchTree[V, T]) Ceil(start, end T) (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V
	interval, ok := ceil(st.root, start, end, st.cmp)
	if !ok {
		return val, false
	}

	return interval.Val, true
}

func ceil[V, T any](root *node[V, T], start, end T, cmp CmpFunc[T]) (interval[V, T], bool) {
	if root == nil {
		return interval[V, T]{}, false
	}

	var ceil *node[V, T]

	cur := root
	for cur != nil {
		if cur.Interval.equal(start, end, cmp) {
			return cur.Interval, true
		}

		if cur.Interval.less(start, end, cmp) {
			cur = cur.Right
		} else {
			ceil = cur
			cur = cur.Left
		}
	}

	if ceil == nil {
		return interval[V, T]{}, false
	}

	return ceil.Interval, true
}

// Floor returns a value which interval key is the greatest interval key lesser than the given start and end interval.
// It returns true as the second return value if there's a floor interval key for the given start and end interval
// in the tree; otherwise, false.
func (st *SearchTree[V, T]) Floor(start, end T) (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V
	interval, ok := floor(st.root, start, end, st.cmp)
	if !ok {
		return val, false
	}

	return interval.Val, true
}

func floor[V, T any](root *node[V, T], start, end T, cmp CmpFunc[T]) (interval[V, T], bool) {
	if root == nil {
		return interval[V, T]{}, false
	}

	var floor *node[V, T]

	cur := root
	for cur != nil {
		if cur.Interval.equal(start, end, cmp) {
			return cur.Interval, true
		}

		if cur.Interval.less(start, end, cmp) {
			floor = cur
			cur = cur.Right
		} else {
			cur = cur.Left
		}
	}

	if floor == nil {
		return interval[V, T]{}, false
	}

	return floor.Interval, true
}

// Rank returns the number of intervals strictly less than the given start and end interval.
func (st *SearchTree[V, T]) Rank(start, end T) int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return rank(st.root, start, end, st.cmp)
}

func rank[V, T any](root *node[V, T], start, end T, cmp CmpFunc[T]) int {
	var rank int
	cur := root

	for cur != nil {
		if cur.Interval.equal(start, end, cmp) {
			rank += size(cur.Left)
			break
		} else if cur.Interval.less(start, end, cmp) {
			rank += 1 + size(cur.Left)
			cur = cur.Right
		} else {
			cur = cur.Left
		}
	}

	return rank
}

// Select returns the value which interval key is the kth smallest interval key in the tree.
// It returns false if k is not between 0 and N-1, where N is the number of interval keys
// in the tree; otherwise, true.
func (st *SearchTree[V, T]) Select(k int) (V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var val V

	interval, ok := selectInterval(st.root, k)
	if !ok {
		return val, false
	}

	return interval.Val, true
}

func selectInterval[V, T any](root *node[V, T], k int) (interval[V, T], bool) {
	cur := root
	for cur != nil {
		t := size(cur.Left)
		switch {
		case t > k:
			cur = cur.Left
		case t < k:
			cur = cur.Right
			k = k - t - 1
		default:
			return cur.Interval, true
		}
	}

	return interval[V, T]{}, false
}

// Find returns the values which interval key exactly matches with the given start and end interval.
// It returns true as the second return value if an exaclty matching interval key is found in the tree;
// otherwise, false.
func (st *MultiValueSearchTree[V, T]) Find(start, end T) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V

	interval, ok := find(st.root, start, end, st.cmp)
	if !ok {
		return vals, false
	}

	return interval.Vals, true
}

// AnyIntersection returns values which interval key intersects with the given start and end interval.
// It returns true as the second return value if any intersection is found in the tree; otherwise, false.
func (st *MultiValueSearchTree[V, T]) AnyIntersection(start, end T) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	interval, ok := anyIntersections(st.root, start, end, st.cmp)
	if !ok {
		return nil, false
	}

	return interval.Vals, true
}

// AllIntersections returns a slice of values which interval key intersects with the given start and end interval.
// It returns true as the second return value if any intersection is found in the tree; otherwise, false.
func (st *MultiValueSearchTree[V, T]) AllIntersections(start, end T) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V
	if st.root == nil {
		return vals, false
	}

	searchInOrder(st.root, start, end, st.cmp, func(it interval[V, T]) {
		vals = append(vals, it.Vals...)
	})

	return vals, len(vals) > 0
}

// Min returns the values which interval key is the minimum interval key in the tree.
// It returns false as the second return value if the tree is empty; otherwise, true.
func (st *MultiValueSearchTree[V, T]) Min() ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V
	if st.root == nil {
		return vals, false
	}

	vals = min(st.root).Interval.Vals

	return vals, true
}

// Max returns the values which interval key is the maximum interval in the tree.
// It returns false as the second return value if the tree is empty; otherwise, true.
func (st *MultiValueSearchTree[V, T]) Max() ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V
	if st.root == nil {
		return vals, false
	}

	vals = max(st.root).Interval.Vals

	return vals, true
}

// Ceil returns the values which interval key is the smallest interval key greater than the given start and end interval.
// It returns true as the second return value if there's a ceiling interval key for the given start and end interval
// in the tree; otherwise, false.
func (st *MultiValueSearchTree[V, T]) Ceil(start, end T) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V
	interval, ok := ceil(st.root, start, end, st.cmp)
	if !ok {
		return vals, false
	}

	return interval.Vals, true
}

// Floor returns the values which interval key is the greatest interval key lesser than the given start and end interval.
// It returns true as the second return value if there's a floor interval key for the given start and end interval
// in the tree; otherwise, false.
func (st *MultiValueSearchTree[V, T]) Floor(start, end T) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V
	interval, ok := floor(st.root, start, end, st.cmp)
	if !ok {
		return vals, false
	}

	return interval.Vals, true
}

// Rank returns the number of intervals strictly less than the given start and end interval.
func (st *MultiValueSearchTree[V, T]) Rank(start, end T) int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return rank(st.root, start, end, st.cmp)
}

// Select returns the values which interval key is the kth smallest interval key in the tree.
// It returns false if k is not between 0 and N-1, where N is the number of interval keys
// in the tree; otherwise, true.
func (st *MultiValueSearchTree[V, T]) Select(k int) ([]V, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	var vals []V

	interval, ok := selectInterval(st.root, k)
	if !ok {
		return vals, false
	}

	return interval.Vals, true
}

// MaxEnd returns the values in the tree that have the largest ending interval.
// It returns false as the second return value if the tree is empty; otherwise, true.
func (st *MultiValueSearchTree[V, T]) MaxEnd() ([]V, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	var vals []V
	if st.root == nil {
		return vals, false
	}

	maxEnd(st.root, st.root.MaxEnd, st.cmp, func(n *node[V, T]) {
		vals = append(vals, n.Interval.Vals...)
	})
	return vals, true
}

func maxEnd[V, T any](n *node[V, T], searchEnd T, cmp CmpFunc[T], visit func(*node[V, T])) {

	// If this node's interval lines up with MaxEnd, visit it.
	if cmp.eq(n.Interval.End, searchEnd) {
		visit(n)
	}

	// Search left if the left subtree contains a max ending interval that is equal to the root's max ending interval.
	if n.Left != nil && cmp.eq(n.Left.MaxEnd, searchEnd) {
		maxEnd(n.Left, searchEnd, cmp, visit)
	}

	// Search right if the right subtree contains a max ending interval that is equal to the root's max ending interval.
	if n.Right != nil && cmp.eq(n.Right.MaxEnd, searchEnd) {
		maxEnd(n.Right, searchEnd, cmp, visit)
	}
}
