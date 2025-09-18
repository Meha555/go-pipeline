package parser

type Dict[K comparable, V any] struct {
	Key   K
	Value V
}

type DictList[K comparable, V any] []Dict[K, V]

func (dl DictList[K, V]) Find(key K) (V, bool) {
	for _, item := range dl {
		if item.Key == key {
			return item.Value, true
		}
	}
	var zero V
	return zero, false
}

func (dl *DictList[K, V]) Prepand(key K, value V) {
	newItem := Dict[K, V]{Key: key, Value: value}
	*dl = append(DictList[K, V]{newItem}, *dl...)
}

func (dl *DictList[K, V]) Append(key K, value V) {
	*dl = append(*dl, Dict[K, V]{
		Key:   key,
		Value: value,
	})
}

func (dl *DictList[K, V]) Merge(other DictList[K, V]) {
	*dl = append(*dl, other...)
}

func (dl DictList[K, V]) ToMap() map[K]V {
	m := make(map[K]V)
	for _, item := range dl {
		m[item.Key] = item.Value
	}
	return m
}
