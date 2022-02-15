/**
 *
 * 2022.1.13 ~
 * author ky
 * 利用数组实现双端队列，主要原因为：温度场计算过程中主要消耗在于遍历计算，因此数组具有更好的局部性，有利于计算速度的提升
 * 该队列的设计主要用于三维温度场计算，因此元素类型为 二维数组
 *
 */

package deque

import "lz/model"

type Deque interface {
	// 队列的长度
	Size() int

	// 获取队列中对应下标的数值
	Get(z, y, x int) float32

	// 获取某个切片
	GetSlice(z int) *model.ItemType

	// 设定队列中对应下标的数值
	Set(z, y, x int, number float32, bottom float32)

	// 正向遍历
	Traverse(f func(z int, item *model.ItemType))

	// 螺旋遍历
	TraverseSpirally(start, end int, f func(z int, item *model.ItemType))

	// 在队列结尾增加一个元素
	AddLast(initialVal float32)

	// 在队列结尾删除一个元素
	RemoveLast()

	// 在队列头部增加一个元素
	AddFirst(initialVal float32)

	// 在队列头部删除一个元素
	RemoveFirst()

	IsFull() bool

	IsEmpty() bool
}
