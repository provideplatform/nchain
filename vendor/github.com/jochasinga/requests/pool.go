// Package requests provide useful and declarative methods for RESTful HTTP requests.
package requests

import "sync"

// Pool represents a variable-sized bufferred channel of type *Response
// which collects results from each request in a goroutine.
type Pool struct {
	Responses    chan *Response
	IgnoreBadURL bool
}

// NewPool creates a *Pool instance with the channel's buffer size of max.
func NewPool(max int) *Pool {
	return &Pool{
		Responses:    make(chan *Response, max),
		IgnoreBadURL: false,
	}
}

// Get sends asychronous GET requests to the provided URLs and returns a receive-only
// channel of type *Response which get closed after all responses are sent to.
func (p *Pool) Get(urls []string, options ...func(*Request)) (<-chan *Response, error) {
	var wg sync.WaitGroup
	output := func(c <-chan *Response) {
		for n := range c {
			p.Responses <- n
		}
		wg.Done()
	}
	wg.Add(len(urls))
	for _, url := range urls {
		rc, err := GetAsync(url, options...)
		if err != nil {
			if p.IgnoreBadURL == false {
				return nil, err
			}
		}
		go output(rc)
	}
	go func() {
		wg.Wait()
		close(p.Responses)
	}()
	return p.Responses, nil
}
