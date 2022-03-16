package deque

import (
	"lz/model"
)

const (
	// 状态
	state0  = 0 // 仅在数组0中存在元素
	state1  = 1 // 仅在数组1中存在元素
	state01 = 2 // 数组0和数组1中均存在元素

	// 数组大小基数
	base = 8
)

type ArrDeque struct {
	// 两个数组一个负责尾部操作，一个负责头部操作
	container  arrStruct
	container1 arrStruct

	// 元素个数
	size int
	// 容量
	capacity int

	// 是否为空
	isEmpty bool

	// 是否满
	isFull bool

	// 状态信息
	state uint8
}

type arrStruct struct {
	arr     ArrType
	start   int
	end     int
	isFull  bool // 该数组是否填充满了，该数组填充满了也代表整个队列达到capacity
	isEmpty bool
}

type ArrType []model.ItemType

// 工厂方法
func NewArrDeque(capacity int) *ArrDeque {
	remainder := capacity % base
	if remainder != 0 {
		capacity = capacity - remainder + base
	}
	arr := make([]model.ItemType, capacity)
	arr1 := make([]model.ItemType, capacity)
	container := arrStruct{
		arr:     arr,
		start:   capacity,
		end:     capacity,
		isFull:  false,
		isEmpty: true,
	}
	container1 := arrStruct{
		arr:     arr1,
		start:   0,
		end:     0,
		isFull:  false,
		isEmpty: true,
	}
	return &ArrDeque{
		container:  container,
		container1: container1,
		size:       0,
		capacity:   capacity,
		isFull:     false,
		isEmpty:    true,
		state:      state0,
	}
}

func (ad *ArrDeque) Size() int {
	return ad.size
}

func (ad *ArrDeque) Get(z, y, x int) float32 {
	l1, l2 := ad.container.end-ad.container.start, ad.container1.end-ad.container1.start
	if z >= l1+l2 {
		panic("index out of length")
	}
	if ad.state == state0 {
		//fmt.Println(z, l1, l2, "Set state0")
		return ad.container.arr[z+ad.container.start][y][x]
	} else if ad.state == state01 {
		//fmt.Println(z, l1, l2, "Set state01")
		if z < l1 {
			return ad.container.arr[z+ad.container.start][y][x]
		}
		return ad.container1.arr[z-l1+ad.container1.start][y][x]
	} else {
		//fmt.Println(z, l1, l2, "Set state1")
		return ad.container1.arr[z+ad.container1.start][y][x]
	}
}

func (ad *ArrDeque) GetSlice(z int) *model.ItemType {
	l1, l2 := ad.container.end-ad.container.start, ad.container1.end-ad.container1.start
	if z >= l1+l2 {
		panic("index out of length")
	}
	if ad.state == state0 {
		//fmt.Println(z, l1, l2, "Set state0")
		return &ad.container.arr[z+ad.container.start]
	} else if ad.state == state01 {
		//fmt.Println(z, l1, l2, "Set state01")
		if z < l1 {
			return &ad.container.arr[z+ad.container.start]
		}
		return &ad.container1.arr[z-l1+ad.container1.start]
	} else {
		//fmt.Println(z, l1, l2, "Set state1")
		return &ad.container1.arr[z+ad.container1.start]
	}
}

func (ad *ArrDeque) Set(z, y, x int, number float32, bottom float32) {
	l1, l2 := ad.container.end-ad.container.start, ad.container1.end-ad.container1.start
	if z >= l1+l2 {
		panic("index out of length")
	}
	if number < bottom {
		number = bottom
	}
	if ad.state == state0 {
		//fmt.Println(z, l1, l2, "Set state0")
		ad.container.arr[z+ad.container.start][y][x] = number
	} else if ad.state == state01 {
		//fmt.Println(z, l1, l2, "Set state01")
		if z < l1 {
			ad.container.arr[z+ad.container.start][y][x] = number
		} else {
			ad.container1.arr[z-l1+ad.container1.start][y][x] = number
		}
	} else {
		//fmt.Println(z, l1, l2, "Set state1")
		ad.container1.arr[z+ad.container1.start][y][x] = number
	}
}

func (ad *ArrDeque) Traverse(f func(z int, item *model.ItemType)) {
	// todo 加入分块
	k := 0
	for z := ad.container.start; z < ad.container.end; z++ {
		f(k, &ad.container.arr[z])
		k++
	}
	for z := ad.container1.start; z < ad.container1.end; z++ {
		f(k, &ad.container1.arr[z])
		k++
	}
	//fmt.Println("Traverse 切片数：", k, ad.container.start, ad.container.end, ad.container1.start, ad.container1.end)
}

func (ad *ArrDeque) TraverseSpirally(start, end int, f func(z int, item *model.ItemType)) {
	l1 := ad.container.end - ad.container.start
	k := start
	if end <= l1 {
		for z := ad.container.start + start; z < ad.container.start+end; z++ {
			f(k, &ad.container.arr[z])
			k++
		}
		return
	}
	l := end - start
	if l1 <= start {
		start -= l1
		for z := ad.container1.start + start; z < ad.container1.start+start+l; z++ {
			f(k, &ad.container1.arr[z])
			k++
		}
		return
	}
	l = ad.container.end - (ad.container.start + start)
	for z := ad.container.start + start; z < ad.container.end; z++ {
		f(k, &ad.container.arr[z])
		k++
	}
	remainder := end - start - l
	for z := ad.container1.start; z < ad.container1.start+remainder; z++ {
		f(k, &ad.container1.arr[z])
		k++
	}
}

func (ad *ArrDeque) AddLast(initialVal float32) {
	if !ad.isFull { // 可能性最大的选项放在最前面
		ad.size++
		if ad.container1.end != ad.capacity { // arr1 end未到最大值
			if ad.container.end == ad.capacity {
				setDefaultVal(&ad.container1.arr[ad.container1.end], initialVal)
				ad.container1.end++
			} else { // removeLast 消耗完了arr1中的元素，并且减到了arr中的元素
				setDefaultVal(&ad.container.arr[ad.container.end], initialVal)
				ad.container.end++
				if ad.container.isEmpty == true {
					ad.container.isEmpty = false
				}
			}
		} else { // arr1 end达到最大值，而且队列未充满，则表示需要更换两个数组
			ad.container, ad.container1 = ad.container1, ad.container // 交换引用
			ad.container1.start, ad.container1.end = 0, 0
			setDefaultVal(&ad.container1.arr[ad.container1.end], initialVal)
			ad.container1.end++
		}
		if ad.container1.isEmpty {
			ad.container1.isEmpty = false
			if !ad.container.isEmpty {
				ad.state = state01
			} else {
				ad.isEmpty = false
				ad.state = state1
			}
		}

		if ad.container1.end-ad.container1.start+ad.container.end-ad.container.start == ad.capacity {
			ad.isFull = true
			if ad.container1.end-ad.container1.start == ad.capacity {
				ad.container1.isFull = true
			}
		}
	} else {
		// todo 扩容
	}
}

func (ad *ArrDeque) RemoveLast() {
	if !ad.isEmpty {
		ad.size--
		if ad.container1.end-1 >= ad.container1.start {
			ad.container1.end--
			if ad.container1.end == ad.container1.start {
				ad.container1.isEmpty = true
				ad.state = state0
				if ad.container.isEmpty {
					ad.isEmpty = true
				}
			}
			if ad.container1.isFull {
				ad.container1.isFull = false
			}
		} else {
			ad.container.end--
			if ad.container.end == ad.container.start {
				ad.container.isEmpty = true
				ad.isEmpty = true
				ad.container.start, ad.container.end = ad.capacity, ad.capacity
				ad.container1.start, ad.container1.end = 0, 0
			}
			if ad.container.isFull {
				ad.container.isFull = false
			}
		}
		if ad.isFull {
			ad.isFull = false
		}
		//fmt.Println(ad.container.end, ad.container.start, ad.container1.end, ad.container1.start, ad.capacity, "RemoveLast")
		// todo 缩容
	}
}

func (ad *ArrDeque) AddFirst(initialVal float32) {
	if !ad.isFull { // 可能性最大的选项放在最前面
		ad.size++
		if ad.container.start != 0 { // arr1的start index变动过
			if ad.container1.start == 0 {
				ad.container.start--
				setDefaultVal(&ad.container.arr[ad.container.start], initialVal)
			} else { // removeFirst 消耗完了arr中的元素，并且减到了arr1中的元素
				ad.container1.start--
				setDefaultVal(&ad.container1.arr[ad.container1.start], initialVal)
				if ad.container1.isEmpty == true {
					ad.container1.isEmpty = false
					ad.state = state1
				}
			}
		} else { // arr start 到达最小位置，并且此时队列未充满，需要交换
			ad.container, ad.container1 = ad.container1, ad.container // 交换引用
			ad.container.start, ad.container.end = ad.capacity, ad.capacity
			ad.container.start--
			setDefaultVal(&ad.container.arr[ad.container.start], initialVal)
		}
		if ad.container.isEmpty {
			ad.container.isEmpty = false
			if !ad.container1.isEmpty {
				ad.state = state01
			} else {
				ad.isEmpty = false
				ad.state = state0
			}
		}
		if ad.container1.end-ad.container1.start+ad.container.end-ad.container.start == ad.capacity {
			ad.isFull = true
			if ad.container.end-ad.container.start == ad.capacity {
				ad.container.isFull = true
			}
		}
		//fmt.Println(ad.container.end, ad.container.start, ad.container1.end, ad.container1.start, ad.capacity, "AddFirst")
	} else {
		// todo 扩容
		//fmt.Println("full...")
	}
}

func (ad *ArrDeque) RemoveFirst() {
	if !ad.isEmpty {
		ad.size--
		if ad.container.start < ad.container.end {
			ad.container.start++
			if ad.container.start == ad.container.end {
				ad.container.isEmpty = true
				if ad.container1.isEmpty {
					ad.isEmpty = true
				} else {
					ad.state = state1
				}
			}
			if ad.container.isFull {
				ad.container.isFull = false
			}
		} else {
			ad.container1.start++
			if ad.container1.end == ad.container1.start {
				ad.container1.isEmpty = true
				ad.isEmpty = true
				ad.container.start, ad.container.end = ad.capacity, ad.capacity
				ad.container1.start, ad.container1.end = 0, 0
			}
			if ad.container1.isFull {
				ad.container1.isFull = false
			}
		}
		if ad.isFull {
			ad.isFull = false
		}
		// todo 缩容
	}
}

func (ad *ArrDeque) IsFull() bool {
	return ad.size == ad.capacity
}

func (ad *ArrDeque) IsEmpty() bool {
	return ad.size == 0
}

func setDefaultVal(item *model.ItemType, initialVal float32) {
	m, n := len(item), len(item[0])
	for y := 0; y < m; y++ {
		for x := 0; x < n; x++ {
			item[y][x] = initialVal
		}
	}
}
