package deque

import (
	"lz/model"
)

type ListDeque struct {
	head *node
	tail *node

	size     int
	capacity int
}

type node struct {
	val  *model.ItemType
	pre  *node
	next *node
}

// 工厂方法
func NewListDeque(capacity int) *ListDeque {
	head := &node{
		val: nil,
	}
	tail := &node{
		val: nil,
	}
	head.next = tail
	tail.pre = head

	return &ListDeque{
		head: head,
		tail: tail,
		size: 0,
		capacity: capacity,
	}
}

func (ld *ListDeque) Size() int {
	return ld.size
}

func (ld *ListDeque) Get(z, y, x int) float32 {
	if z > ld.size {
		panic("index out of length")
	}
	iter := &node{}
	iter = ld.head.next
	for i := 0; i < z; i++ {
		iter = iter.next
	}
	return iter.val[y][x]
}

func (ld *ListDeque) GetSlice(z int) *model.ItemType {
	if z > ld.size {
		panic("index out of length")
	}
	iter := &node{}
	iter = ld.head.next
	for i := 0; i < z; i++ {
		iter = iter.next
	}
	return iter.val
}

func (ld *ListDeque) Set(z, y, x int, number float32, bottom float32) {
	if z > ld.size {
		panic("index out of length")
	}
	iter := &node{}
	iter = ld.head.next
	for i := 0; i < z; i++ {
		iter = iter.next
	}
	if number <= bottom {
		number = bottom
	}
	iter.val[y][x] = number
}

func (ld *ListDeque) Traverse(f func(z int, item *model.ItemType), start int, end int) {
	iter := &node{}
	z := 0
	for iter = ld.head.next; iter != ld.tail; iter = iter.next {
		f(z, iter.val)
		z++
	}
	//fmt.Println(z, "Traverse")
}

func (ld *ListDeque) TraverseSpirally(start, end int, f func(z int, item *model.ItemType)) {
	iter := &node{}
	iter = ld.head
	for i := 0; i <= start ; i++ {
		iter = iter.next
	}
	for z := start; z < end; z++ {
		f(z, iter.val)
		iter = iter.next
	}
}

func (ld *ListDeque) AddLast(initialVal float32) {
	if ld.IsFull() {
		return
	}
	item := &model.ItemType{}
	m, n := len(item), len(item[0])
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			item[i][j] = initialVal
		}
	}
	newNode := &node{
		val: item,
	}
	tmp := ld.tail.pre
	ld.tail.pre = newNode
	newNode.next = ld.tail
	newNode.pre = tmp
	tmp.next = newNode
	ld.size++
}

func (ld *ListDeque) RemoveLast() {
	if ld.size > 0 {
		ld.tail.pre = ld.tail.pre.pre
		ld.tail.pre.next = ld.tail
		ld.size--
	}
}

func (ld *ListDeque) AddFirst(initialVal float32) {
	if ld.IsFull() {
		return
	}
	item := &model.ItemType{}
	m, n := len(item), len(item[0])
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			item[i][j] = initialVal
		}
	}
	newNode := &node{
		val: item,
	}
	tmp := ld.head.next
	ld.head.next = newNode
	newNode.pre = ld.head
	newNode.next = tmp
	tmp.pre = newNode
	ld.size++
}

func (ld *ListDeque) RemoveFirst() {
	if ld.size > 0 {
		ld.head.next = ld.head.next.next
		ld.head.next.pre = ld.head
		ld.size--
	}
}

func (ld *ListDeque) IsFull() bool {
	return ld.size == ld.capacity
}

func (ld *ListDeque) IsEmpty() bool {
	return ld.size == 0
}