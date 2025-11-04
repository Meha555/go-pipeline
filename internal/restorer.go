package internal

import (
	"container/list"
	"errors"
)

type Stack struct {
	items *list.List
}

func NewStack() *Stack {
	return &Stack{
		items: list.New(),
	}
}

func (r *Stack) Push(item interface{}) {
	r.items.PushBack(item)
}

func (r *Stack) Pop() (interface{}, error) {
	if r.items.Len() == 0 {
		return nil, errors.New("stack is empty")
	}
	return r.items.Remove(r.items.Back()), nil
}
