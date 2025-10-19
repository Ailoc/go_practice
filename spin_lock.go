package main

import "sync/atomic"

type SpinLock struct {
	flag int32
}

func (sl *SpinLock) Lock() {
	for !atomic.CompareAndSwapInt32(&sl.flag, 0, 1) {
		// 自旋等待
	}
}

func (sl *SpinLock) Unlock() {
	atomic.StoreInt32(&sl.flag, 0)
}
