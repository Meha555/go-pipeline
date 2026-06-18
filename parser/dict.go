package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Dict[K comparable, V any] struct {
	Key   K
	Value V
}

type DictList[K comparable, V any] []Dict[K, V]

func (dl *DictList[K, V]) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping, got %s", value.ShortTag())
	}
	items := make(DictList[K, V], 0, len(value.Content)>>1)
	for i := 0; i < len(value.Content); i += 2 {
		var key K
		if err := value.Content[i].Decode(&key); err != nil {
			return err
		}
		var val V
		if err := value.Content[i+1].Decode(&val); err != nil {
			return err
		}
		items = append(items, Dict[K, V]{Key: key, Value: val})
	}
	*dl = items
	return nil
}

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
