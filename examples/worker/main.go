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
	count = flag.Int("count", 10, "for setting size of tasks")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	jobs := make(chan workers.Job[*result])
	worker := workers.NewWorker(jobs)
	worker.Start(ctx)
	defer worker.Close()

	resp := make(chan *result)
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		for i := 0; i < *count; i++ {
			j := &job{
				method: http.MethodGet,
				addr:   "https://google.com",
				body:   http.NoBody,
				res:    resp,
				client: &http.Client{},
			}
			jobs <- j
			log.Printf("send element %d", i)
		}
	}()

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
