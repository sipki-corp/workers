package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/sipki-corp/workers"
)

var _ workers.Job[*http.Response] = &job{}

type job struct {
	method string
	addr   string
	body   io.Reader
	res    chan workers.Result[*http.Response]
	client *http.Client
}

func (j *job) Do(ctx context.Context) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, j.method, j.addr, j.body)
	if err != nil {
		return nil, err
	}

	time.Sleep(time.Second)

	return j.client.Do(req)
}

func (j *job) Result() chan<- workers.Result[*http.Response] {
	return j.res
}

var (
	count = flag.Int("count", 100, "for setting size of tasks")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := workers.NewPool[*http.Response]()
	if err != nil {
		panic(err)
	}

	pool.Start(ctx)
	defer pool.Close()
	log.Printf("max pool worker size: %d", pool.Max())

	currentSize, err := pool.WorkerSize(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("current pool worker size: %d", currentSize)

	jobSize, err := pool.JobBufferSize(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("current job queue size: %d", jobSize)

	pool.Resize(10)
	currentSize, err = pool.WorkerSize(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("current pool worker size: %d", currentSize)

	pool.Resize(-5)
	currentSize, err = pool.WorkerSize(ctx)
	if err != nil {
		panic(err)
	}

	pool.Resize(-50)
	currentSize, err = pool.WorkerSize(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("current pool worker size: %d", currentSize)

	pool.Resize(10)
	currentSize, err = pool.WorkerSize(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("current pool worker size: %d", currentSize)

	resp := make(chan workers.Result[*http.Response])
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()

	go func() {
		defer wg.Done()
		for i := 0; i < *count; i++ {
			pool.Send(&job{
				method: http.MethodGet,
				addr:   "https://google.com",
				body:   http.NoBody,
				res:    resp,
				client: &http.Client{},
			},
			)

			log.Printf("send element %d", i)
		}
	}()

	time.Sleep(time.Second * 2)
	jobSize, err = pool.JobBufferSize(ctx)
	if err != nil {
		panic(err)
	}
	log.Printf("current job queue size: %d", jobSize)

	results := 0
	for {
		if results == *count {
			break
		}

		select {
		case result := <-resp:
			if result.Err != nil {
				panic(result.Err)
			}

			closeErr := result.Value.Body.Close()
			if closeErr != nil {
				panic(closeErr)
			}

			log.Println("success result")
			results++

		case <-ctx.Done():
			return
		}
	}
}
