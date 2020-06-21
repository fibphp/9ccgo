package main

import (
	"fmt"
	"testing"
)

func Test_Cache(t *testing.T) {
	lru := LRUNew(5)
	lru.Add("1111", 1111)
	lru.Add("2222", 2222)
	lru.Add("3333", 3333)
	lru.Add("4444", 4444)
	lru.Add("5555", 5555)
	lru.Print()
	//get成功后，可以看到3333元素移动到了链表头
	fmt.Println(lru.Get("3333"))
	lru.Print()
	//再次添加元素，如果超过最大数量，则删除链表尾部元素，将新元素添加到链表头
	lru.Add("6666", 6666)
	lru.Print()
	lru.Del("4444")
	lru.Print()
	lru.Set("2222", "242424")
	lru.Print()
	fmt.Println(lru.Get("4444"))
	fmt.Println(lru.Get("2222"))

	lru0 := LRUNew(0)
	lru0.Print()
	fmt.Println(lru0.Get("4444"))
	lru.Set("2222", "242424")
	lru0.Print()
	fmt.Println(lru0.Get("2222"))
}
