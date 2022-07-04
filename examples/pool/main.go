package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/sipki-corp/workers"
)

type result struct {
	resp *http.Response
	err  error
}

var _ workers.Job[*result] = &job{}

type job struct {
	method string
	addr   string
	body   io.Reader
	res    chan *result
	client *http.Client
}

func (j *job) Do(ctx context.Context) *result {
	req, err := http.NewRequestWithContext(ctx, j.method, j.addr, j.body)
	if err != nil {
		return &result{err: err}
	}

	resp, err := j.client.Do(req)

	return &result{resp: resp, err: err}
}

func (j *job) Result() chan<- *result {
	return j.res
}

var (
	count = flag.Int("count", 100, "for setting size of tasks")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := workers.NewPool[*result]()
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
	log.Printf("current pool worker size: %d", currentSize)

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

	resp := make(chan *result)

	for i := 1; i <= *count; i++ {
		j := &job{
			method: http.MethodGet,
			addr:   "https://google.com",
			body:   http.NoBody,
			res:    resp,
			client: &http.Client{},
		}

		pool.Publish(j)

		log.Printf("send element %d", i)

		jobSize, err := pool.JobBufferSize(ctx)
		if err != nil {
			panic(err)
		}
		log.Printf("current job queue size: %d", jobSize)
	}

	results := 0
	for {
		if results == *count {
			break
		}

		select {
		case result := <-resp:
			if result.err != nil {
				panic(result.err)
			}

			closeErr := result.resp.Body.Close()
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
