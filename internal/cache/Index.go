package cache

import (
	"sync"
	"time"
)

type val struct {
	data        interface{}
	expiredTime int64
}

const delChannelCap = 100

type ExpiredMap struct {
	m       map[interface{}]*val
	timeMap map[int64][]interface{}
	lck     *sync.Mutex
	stop    chan struct{}
}

func NewExpiredMap() *ExpiredMap {
	e := ExpiredMap{
		m:       make(map[interface{}]*val),
		lck:     new(sync.Mutex),
		timeMap: make(map[int64][]interface{}),
		stop:    make(chan struct{}),
	}
	go e.run(time.Now().Unix())
	return &e
}

type delMsg struct {
	keys []interface{}
	t    int64
}

// background goroutine 主动删除过期的key
// 数据实际删除时间比应该删除的时间稍晚一些，这个误差会在查询的时候被解决。
func (e *ExpiredMap) run(now int64) {
	t := time.NewTicker(time.Second * 1)
	defer t.Stop()
	delCh := make(chan *delMsg, delChannelCap)
	go func() {
		for v := range delCh {
			e.multiDelete(v.keys, v.t)
		}
	}()
	for {
		select {
		case <-t.C:
			now++ //这里用now++的形式，直接用time.Now().Unix()可能会导致时间跳过1s，导致key未删除。
			e.lck.Lock()
			if keys, found := e.timeMap[now]; found {
				e.lck.Unlock()
				delCh <- &delMsg{keys: keys, t: now}
			} else {
				e.lck.Unlock()
			}
		case <-e.stop:
			close(delCh)
			return
		}
	}
}

func (e *ExpiredMap) Set(key, value interface{}, expireSeconds int64) {
	if expireSeconds <= 0 {
		return
	}
	e.lck.Lock()
	defer e.lck.Unlock()
	expiredTime := time.Now().Unix() + expireSeconds
	e.m[key] = &val{
		data:        value,
		expiredTime: expiredTime,
	}
	e.timeMap[expiredTime] = append(e.timeMap[expiredTime], key) //过期时间作为key，放在map中
}

func (e *ExpiredMap) Get(key interface{}) (found bool, value interface{}) {
	e.lck.Lock()
	defer e.lck.Unlock()
	if found = e.checkDeleteKey(key); !found {
		return
	}
	value = e.m[key].data
	return
}

func (e *ExpiredMap) Delete(key interface{}) {
	e.lck.Lock()
	delete(e.m, key)
	e.lck.Unlock()
}

func (e *ExpiredMap) Remove(key interface{}) {
	e.Delete(key)
}

func (e *ExpiredMap) multiDelete(keys []interface{}, t int64) {
	e.lck.Lock()
	defer e.lck.Unlock()
	delete(e.timeMap, t)
	for _, key := range keys {
		delete(e.m, key)
	}
}

func (e *ExpiredMap) Length() int { //结果是不准确的，因为有未删除的key
	e.lck.Lock()
	defer e.lck.Unlock()
	return len(e.m)
}

func (e *ExpiredMap) Size() int {
	return e.Length()
}

// 返回key的剩余生存时间 key不存在返回负数
func (e *ExpiredMap) TTL(key interface{}) int64 {
	e.lck.Lock()
	defer e.lck.Unlock()
	if !e.checkDeleteKey(key) {
		return -1
	}
	return e.m[key].expiredTime - time.Now().Unix()
}

func (e *ExpiredMap) Clear() {
	e.lck.Lock()
	defer e.lck.Unlock()
	e.m = make(map[interface{}]*val)
	e.timeMap = make(map[int64][]interface{})
}

func (e *ExpiredMap) Close() { // todo 关闭后在使用怎么处理
	e.lck.Lock()
	defer e.lck.Unlock()
	e.stop <- struct{}{}
	//e.m = nil
	//e.timeMap = nil
}

func (e *ExpiredMap) Stop() {
	e.Close()
}

func (e *ExpiredMap) DoForEach(handler func(interface{}, interface{})) {
	e.lck.Lock()
	defer e.lck.Unlock()
	for k, v := range e.m {
		if !e.checkDeleteKey(k) {
			continue
		}
		handler(k, v)
	}
}

func (e *ExpiredMap) DoForEachWithBreak(handler func(interface{}, interface{}) bool) {
	e.lck.Lock()
	defer e.lck.Unlock()
	for k, v := range e.m {
		if !e.checkDeleteKey(k) {
			continue
		}
		if handler(k, v) {
			break
		}
	}
}

func (e *ExpiredMap) checkDeleteKey(key interface{}) bool {
	if val, found := e.m[key]; found {
		if val.expiredTime <= time.Now().Unix() {
			delete(e.m, key)
			//delete(e.timeMap, val.expiredTime)
			return false
		}
		return true
	}
	return false
}
