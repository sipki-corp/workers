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

	return j.client.Do(req)
}

func (j *job) Result() chan<- workers.Result[*http.Response] {
	return j.res
}

var (
	count = flag.Int("count", 10, "for setting size of tasks")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	jobs := make(chan workers.Job[*http.Response])
	worker := workers.NewWorker(jobs)
	worker.Start(ctx)
	defer worker.Close()

	resp := make(chan workers.Result[*http.Response])
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		for i := 0; i < *count; i++ {
			jobs <- &job{
				method: http.MethodGet,
				addr:   "https://google.com",
				body:   http.NoBody,
				res:    resp,
				client: &http.Client{},
			}
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
