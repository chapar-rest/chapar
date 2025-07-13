package interval

// Delete removes the given start and end interval key and its associated value from the tree.
// It does nothing if the given start and end interval key doesn't exist in the tree.
//
// Delete returns an InvalidIntervalError if the given end is less than or equal to the given start value.
func (st *SearchTree[V, T]) Delete(start, end T) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.root == nil {
		return nil
	}

	intervl := interval[V, T]{
		Start:      start,
		End:        end,
		AllowPoint: st.config.allowIntervalPoint,
	}

	if intervl.isInvalid(st.cmp) {
		return newInvalidIntervalError(intervl)
	}

	st.root = delete(st.root, intervl, st.cmp)
	if st.root != nil {
		st.root.Color = black
	}

	return nil
}

func delete[V, T any](n *node[V, T], intervl interval[V, T], cmp CmpFunc[T]) *node[V, T] {
	if n == nil {
		return nil
	}

	if intervl.less(n.Interval.Start, n.Interval.End, cmp) {
		if n.Left != nil && !isRed(n.Left) && !isRed(n.Left.Left) {
			n = moveRedLeft(n, cmp)
		}
		n.Left = delete(n.Left, intervl, cmp)
	} else {
		if isRed(n.Left) {
			n = rotateRight(n, cmp)
		}
		if n.Interval.equal(intervl.Start, intervl.End, cmp) && n.Right == nil {
			return nil
		}
		if n.Right != nil && !isRed(n.Right) && !isRed(n.Right.Left) {
			n = moveRedRight(n, cmp)
		}
		if n.Interval.equal(intervl.Start, intervl.End, cmp) {
			minNode := min(n.Right)
			n.Interval = minNode.Interval
			n.Right = deleteMin(n.Right, cmp)
		} else {
			n.Right = delete(n.Right, intervl, cmp)
		}
	}

	updateSize(n)

	return fixUp(n, cmp)
}

func deleteMin[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	if n.Left == nil {
		return nil
	}

	if !isRed(n.Left) && !isRed(n.Left.Left) {
		n = moveRedLeft(n, cmp)
	}

	n.Left = deleteMin(n.Left, cmp)

	updateSize(n)

	return fixUp(n, cmp)
}

// DeleteMin removes the smallest interval key and its associated value from the tree.
func (st *SearchTree[V, T]) DeleteMin() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.root == nil {
		return
	}

	st.root = deleteMin(st.root, st.cmp)
	if st.root != nil {
		st.root.Color = black
	}
}

// DeleteMax removes the largest interval key and its associated value from the tree.
func (st *SearchTree[V, T]) DeleteMax() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.root == nil {
		return
	}

	st.root = deleteMax(st.root, st.cmp)
	if st.root != nil {
		st.root.Color = black
	}
}

func deleteMax[V, T any](n *node[V, T], cmp CmpFunc[T]) *node[V, T] {
	if isRed(n.Left) {
		n = rotateRight(n, cmp)
	}

	if n.Right == nil {
		return nil
	}

	if !isRed(n.Right) && !isRed(n.Right.Left) {
		n = moveRedRight(n, cmp)
	}

	n.Right = deleteMax(n.Right, cmp)

	updateSize(n)

	return fixUp(n, cmp)
}

// Delete removes the given start and end interval key and its associated values from the tree.
// It does nothing if the given start and end interval key doesn't exist in the tree.
//
// Delete returns an InvalidIntervalError if the given end is less than or equal to the given start value.
func (st *MultiValueSearchTree[V, T]) Delete(start, end T) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.root == nil {
		return nil
	}

	intervl := interval[V, T]{
		Start:      start,
		End:        end,
		AllowPoint: st.config.allowIntervalPoint,
	}

	if intervl.isInvalid(st.cmp) {
		return newInvalidIntervalError(intervl)
	}

	st.root = delete(st.root, intervl, st.cmp)
	if st.root != nil {
		st.root.Color = black
	}

	return nil
}

// DeleteMin removes the smallest interval key and its associated values from the tree.
func (st *MultiValueSearchTree[V, T]) DeleteMin() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.root == nil {
		return
	}

	st.root = deleteMin(st.root, st.cmp)
	if st.root != nil {
		st.root.Color = black
	}
}

// DeleteMax removes the largest interval key and its associated values from the tree.
func (st *MultiValueSearchTree[V, T]) DeleteMax() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.root == nil {
		return
	}

	st.root = deleteMax(st.root, st.cmp)
	if st.root != nil {
		st.root.Color = black
	}
}
