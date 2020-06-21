package utils

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type Payload interface {
	Play(w *Worker)
	Name() string
}

type Job struct {
	Payload Payload
}

//  工人
type Worker struct {
	name        string //工人的名字
	jobChannel  chan *Job
	stopChannel chan bool
	Dispatcher  *Dispatcher
	isRunning   bool
}

type Dispatcher struct {
	name        string
	WorkerList  []*Worker
	maxWorkers  int
	jobChanPool chan *Worker
	JobQueue    chan *Job
	QuitChan    chan struct{}
	isRunning   bool
	isStopped   bool
	wg          sync.WaitGroup
	debug       bool
}

func GetGID() int {
	defer func()  {
		if err := recover(); err != nil {
			fmt.Printf("panic recover:panic info:%v\n", err)
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

// 新建一个工人
func NewWorker(name string, dispatcher *Dispatcher) *Worker {

	if dispatcher.debug {
		fmt.Printf("[%s] 工人初始化 \n", name)
	}
	return &Worker{
		name:        name,
		jobChannel:  make(chan *Job),
		stopChannel: make(chan bool),
		Dispatcher:  dispatcher,
		isRunning:   false,
	}
}
func (w *Worker) Stop() {
	w.stopChannel <- true
}

func (w *Worker) Name() string {
	return w.name
}

// 工人开始工作
func (w *Worker) Start() {
	if w.isRunning {
		return
	}

	go func() {
		w.isRunning = true
		for {
			w.Dispatcher.jobChanPool <- w
			if w.Dispatcher.debug {
				fmt.Printf("[%s] 注册到空闲池中 \n", w.name)
			}
			select {
			case job := <-w.jobChannel:
				if w.Dispatcher.debug {
					fmt.Printf("[%s] 接到任务 [%s] 当前空闲:%d \n", w.name, job.Payload.Name(), len(w.Dispatcher.jobChanPool))
				}
				job.Payload.Play(w)
				w.Dispatcher.wg.Done()
			case <-w.Dispatcher.QuitChan:
				w.isRunning = false
				return
			case <-w.stopChannel:
				w.isRunning = false
				return
			}
		}
	}()
}

func NewDispatcher(name string, maxWorkers int, jobQueue chan *Job, debug bool) *Dispatcher {
	dispatcher := &Dispatcher{
		WorkerList:  make([]*Worker, maxWorkers),
		jobChanPool: make(chan *Worker, maxWorkers),
		JobQueue:    jobQueue,
		QuitChan:    make(chan struct{}),
		name:        name,
		maxWorkers:  maxWorkers,
		isRunning:   false,
		isStopped:   false,
		debug:       debug,
	}

	for i := 0; i < maxWorkers; i++ {
		wName := fmt.Sprintf("%s:work-%d", name, i)
		dispatcher.WorkerList[i] = NewWorker(wName, dispatcher)
	}
	return dispatcher
}

func (d *Dispatcher) Run() {
	if d.isStopped {
		return
	}

	for _, worker := range d.WorkerList {
		if !worker.isRunning {
			worker.Start()
		}
	}
	if d.isRunning {
		return
	}

	go func() {
		d.isRunning = true
		for {
			select {
			case job := <-d.JobQueue:
				if job != nil {
					d.wg.Add(1)
					go func(job *Job) {
						for {
							worker := <-d.jobChanPool
							if worker.isRunning {
								worker.jobChannel <- job
								return
							}
						}
					}(job)
				}
			case <-d.QuitChan:
				d.isRunning = false
				return
			}
		}
	}()
}

func (d *Dispatcher) Join() {
	if d.isStopped {
		return
	}
	d.wg.Wait()
}

func (d *Dispatcher) Stop() {
	d.isStopped = true
	close(d.QuitChan)
}
