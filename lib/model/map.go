package model

import "iter"

func NewLinkedMap[K comparable, V any]() LinkedMap[K, V] {
	return LinkedMap[K, V]{}
}

type LinkedMap[K comparable, V any] struct {
	m     map[K]*mapEntry[K, V]
	start *mapEntry[K, V]
	end   *mapEntry[K, V]
}

type mapEntry[K comparable, V any] struct {
	k    K
	v    V
	prev *mapEntry[K, V]
	next *mapEntry[K, V]
}

func (o *LinkedMap[K, V]) Put(k K, v V) {
	if o.m == nil {
		o.m = make(map[K]*mapEntry[K, V])
	}

	// key exists, updated value
	if _, inMap := o.m[k]; inMap {
		o.m[k].v = v
		return
	}

	entry := &mapEntry[K, V]{k: k, v: v}
	o.m[k] = entry

	if o.end == nil {
		// list is empty
		o.end = entry
		o.start = entry
	} else {
		// append to the end
		o.end.next = entry
		entry.prev = o.end
		o.end = entry
	}
}

func (o *LinkedMap[K, V]) GetVal(k K) V {
	if o.m != nil {
		if n, ok := o.m[k]; ok {
			return n.v
		}
	}
	var zero V
	return zero
}

func (o *LinkedMap[K, V]) Get(k K) (V, bool) {
	if o.m != nil {
		if n, ok := o.m[k]; ok {
			return n.v, true
		}
	}
	var zero V // Return zero value for type V.
	return zero, false
}

func (o *LinkedMap[K, V]) Remove(k K) {
	if o.m == nil {
		return
	}

	if entry, ok := o.m[k]; ok {
		delete(o.m, k)

		// Unlink the node from the list
		if entry.prev != nil {
			entry.prev.next = entry.next
		} else {
			// entry is at start
			o.start = entry.next
		}

		if entry.next != nil {
			entry.next.prev = entry.prev
		} else {
			// entry is the end
			o.end = entry.prev
		}
	}
}

func (o *LinkedMap[K, V]) Keys() []K {
	keys := make([]K, 0, len(o.m))
	for entry := o.start; entry != nil; entry = entry.next {
		keys = append(keys, entry.k)
	}
	return keys
}

func (o *LinkedMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(key K, value V) bool) {
		for n := o.start; n != nil; n = n.next {
			if !yield(n.k, n.v) {
				return
			}
		}
	}
}
