package utils

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
)

//LRU(Least recently used)最近最少使用，算法根据数据的历史访问记录来进行淘汰数据
//核心思想是"如果数据最近被访问过，那么将来被访问的几率也更高"
//常见的实现方式是用一个链表保存数据
//1. 新数据插入到链表头部
//2. 每当缓存命中(即缓存数据被访问)，则将数据移到链表头部
//3. 当链表满的时候，将链表尾部的数据丢弃

type cacheItem struct {
	Key string
	Val interface{}
}

type LRU struct {
	//最大存储数量
	maxNum int
	//当前存储数量
	curNum int
	//锁，保证数据一致性
	mutex sync.Mutex
	//链表
	data *list.List
}

//添加数据
func (l *LRU) Add(key string, value interface{}) error {
	//判断key是否存在
	if e, _ := l.Exist(key); e {
		return errors.New(key + "已存在")
	}
	//判断当前存储数量与最大存储数量
	if l.maxNum == l.curNum {
		//链表已满，则删除链表尾部元素
		l.clear()
	}
	l.mutex.Lock()
	l.curNum++
	data := cacheItem{key, value}
	//把数据保存到链表头部
	l.data.PushFront(data)
	l.mutex.Unlock()
	return nil
}

//设置数据
func (l *LRU) Set(key string, value interface{}) error {
	e, item := l.Exist(key)
	if !e {
		return l.Add(key, value)
	}
	data := cacheItem{key, value}

	l.mutex.Lock()
	//设置链表元素数据
	item.Value = data
	l.mutex.Unlock()
	return nil
}

//清理数据
func (l *LRU) clear() interface{} {
	l.mutex.Lock()
	l.curNum--
	//删除链表尾部元素
	v := l.data.Remove(l.data.Back())
	l.mutex.Unlock()
	return v
}

//获取数据
func (l *LRU) Get(key string) interface{} {
	e, item := l.Exist(key)
	if !e {
		return nil
	}
	l.mutex.Lock()
	//数据被访问，则把元素移动到链表头部
	l.data.MoveToFront(item)
	l.mutex.Unlock()
	data := item.Value.(cacheItem)
	return data.Val
}

//删除数据
func (l *LRU) Del(key string) error {
	e, item := l.Exist(key)
	if !e {
		return errors.New(key + "不存在")
	}
	l.mutex.Lock()
	l.curNum--
	//删除链表元素
	l.data.Remove(item)
	l.mutex.Unlock()
	return nil
}

//判断是否存在
func (l *LRU) Exist(key string) (bool, *list.Element) {
	//循环链表，判断key是否存在
	for v := l.data.Front(); v != nil; v = v.Next() {
		data := v.Value.(cacheItem)
		if key == data.Key {
			return true, v
		}
	}
	return false, nil
}

//返回长度
func (l *LRU) Len() int {
	return l.curNum
}

//打印链表
func (l *LRU) Print() {
	fmt.Println("length:", l.Len())
	for v := l.data.Front(); v != nil; v = v.Next() {
		data := v.Value.(cacheItem)
		fmt.Println("key:", data.Key, " value:", data.Val)
	}
}

//创建一个新的LRU
func LRUNew(maxNum int) *LRU {
	return &LRU{
		maxNum: maxNum,
		curNum: 0,
		mutex:  sync.Mutex{},
		data:   list.New(),
	}
}
