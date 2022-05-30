package workerpool

import (
	"runtime/trace"
	"sync"
)

type Pool struct {
	capacity int    		// workerpool 大小

	active chan struct{}	// active channel
	task chan trace.Task

	wg sync.WaitGroup		// 用于在pool销毁时等待所有worker退出
	quit chan struct{}		// 用于通知各个worker退出的信号channel
}
