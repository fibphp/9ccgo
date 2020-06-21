package utils

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

type TestPayload struct {
	name string
}

func (p *TestPayload) Name() string {
	return p.name
}
func (p *TestPayload) Play(w *Worker) {
	sn := rand.Int31n(1000)
	time.Sleep(time.Duration(sn) * time.Millisecond)
	time.Sleep(time.Second)
	fmt.Printf("[%s] sleep %d ms ...完成 by [%s] \n", p.name, sn, w.name)
}

func Test_main(t *testing.T) {
	jobQueue := make(chan *Job)
	dispatch := NewDispatcher("test", 8, jobQueue, true)
	dispatch.Run()
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("job-%s", strconv.Itoa(i))
		fmt.Printf("[%s] 初始化新任务 \n", name)
		p := &TestPayload{
			name: name,
		}
		jobQueue <- &Job{
			Payload: p,
		}
		time.Sleep(time.Second * time.Duration(rand.Int31n(5)))
	}

	dispatch.Join()

	for i := 10; i < 40; i++ {
		name := fmt.Sprintf("job-%s", strconv.Itoa(i))
		fmt.Printf("[%s] 初始化新任务 \n", name)
		p := &TestPayload{
			name: name,
		}
		jobQueue <- &Job{
			Payload: p,
		}
		//time.Sleep(time.Second)
	}

	dispatch.Join()
	dispatch.Stop()
	close(jobQueue)
}
