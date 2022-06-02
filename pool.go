package workerpool

import (
	"errors"
	"fmt"
	"sync"
)

var defaultCapacity = 100	// 默认池大小
var maxCapacity = 2000 		// 最大池大小

type Task func()

type Pool struct {
	capacity int    		// workerpool 大小

	preAlloc bool			// 是否在创建 pool 的时候就预创建workers，默认值: false
	// 当pool 满的情况下，新的 Schedule 调用是否阻塞当前goroutine，默认值: true
	// 如果 block = false，则 Schedule 返回 ErrNoWorkerAvailInPool
	block bool

	active chan struct{}	// active channel
	tasks chan Task

	wg sync.WaitGroup		// 用于在pool销毁时等待所有worker退出
	quit chan struct{}		// 用于通知各个worker退出的信号channel
}

func New(capacity int, opts ...Option) *Pool {
	if capacity < 0 {
		capacity = defaultCapacity
	}
	if capacity > maxCapacity {
		capacity = maxCapacity
	}

	p := &Pool{
		capacity: capacity,

		active: make(chan struct{}, capacity),
		tasks: make(chan Task),
		quit: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(p)
	}
	fmt.Printf("workerpool start(preAlloc=%t)\n", p.preAlloc)

	if p.preAlloc {
		for i := 0; i < capacity; i++ {
			// create all goroutines and send into works channel
			p.newWorker(i + 1)
			p.active <- struct{}{}
		}
	}

	go p.run()

	return p
}

func (p *Pool) run() {
	idx := len(p.active)
	if !p.preAlloc {
	loop:
		for t := range p.tasks {
			p.returnTask(t)
			select {
			case <-p.quit:
				return
			case p.active <- struct{}{}:
				idx++
				p.newWorker(idx)
			default:
				break loop
			}
		}
	}

	for {
		select {
		case <-p.quit:
			return
		case p.active <- struct{}{}:
			// 创建 worker
			idx++
			p.newWorker(idx)
		}
	}
}

func (p *Pool) newWorker(i int) {
	p.wg.Add(1)
	go func() {
		// 消除 panic，不让其向上传到到调用函数
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("worker[%03d]: revover panic[%s] and exit\n", i, err)
				<-p.active
			}
			p.wg.Done()
		}()

		fmt.Printf("worker[%03d]: start\n", i)

		for {
			select {
			case <-p.quit:
				fmt.Printf("worker[%03d]: exit\n", i)
				<-p.active
				return
			case t := <-p.tasks:
				fmt.Printf("worker[%03d]: receive a task\n", i)
				t()
			}
		}
	}()

}

var ErrWorkerPoolFreed = errors.New("no idle worker in pool") // worker pool 释放

func (p *Pool) Schedule(t Task) error {
	select {
	case <-p.quit:
		return ErrWorkerPoolFreed
	case p.tasks <- t:
		return nil
	default:
		if p.block {
			p.tasks <- t
			return nil
		}
		return ErrWorkerPoolFreed

	}
}

func (p * Pool) Free() {

	// p.quit <- struct{}{} 我开始使用的是这个，
	// 结果会死锁，我想大概是和java中的signal和signalAll的区别吧
	close(p.quit)

	p.wg.Wait()

	fmt.Printf("workerpool freed(preAlloc=%t)\n", p.preAlloc)
}

func (p *Pool) returnTask(task Task)  {

}

