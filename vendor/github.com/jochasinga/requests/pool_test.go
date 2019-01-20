// Package requests provide useful and declarative methods for RESTful HTTP requests.
package requests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPoolConstructor(t *testing.T) {
	p := NewPool(10)
	result := reflect.TypeOf(p)
	expected := reflect.TypeOf(&Pool{})
	if result != expected {
		t.Error(unexpectErr(result, expected))
	}
	if p.Responses == nil {
		t.Error("Responses channel not created.")
	}
}

var (
	ts1 = httptest.NewServer(http.HandlerFunc(helloHandler))
	ts2 = httptest.NewServer(http.HandlerFunc(jsonHandler))
	ts3 = httptest.NewServer(http.HandlerFunc(htmlHandler))
)

func TestPoolGetResponseType(t *testing.T) {
	p := NewPool(1)
	rc, err := p.Get([]string{ts1.URL})
	if err != nil {
		t.Error(err)
	}
	result := reflect.TypeOf(rc)
	expected := reflect.TypeOf((<-chan *Response)(nil))
	if result != expected {
		t.Error(unexpectErr(result, expected))
	}
}

func TestPoolGetURLs(t *testing.T) {
	p := NewPool(3)
	rc, err := p.Get([]string{ts1.URL, ts2.URL, ts3.URL})
	if err != nil {
		t.Error(err)
	}
	resp1 := <-rc
	if resp1.StatusCode != 200 {
		t.Error(unexpectErr(resp1.StatusCode, 200))
	}
	resp2 := <-rc
	if resp2.StatusCode != 200 {
		t.Error(unexpectErr(resp2.StatusCode, 404))
	}
	resp3 := <-rc
	if resp3.StatusCode != 200 {
		t.Error(unexpectErr(resp3.StatusCode, 200))
	}
}

func TestPoolGetSameURL(t *testing.T) {
	p := NewPool(3)
	urls := []string{}
	for i := 0; i <= 3; i++ {
		urls = append(urls, ts1.URL)
	}
	rc, err := p.Get(urls)
	if err != nil {
		t.Error(err)
	}
	resp1 := <-rc
	if resp1.StatusCode != 200 {
		t.Error(unexpectErr(resp1.StatusCode, 200))
	}
	resp2 := <-rc
	if resp2.StatusCode != 200 {
		t.Error(unexpectErr(resp2.StatusCode, 200))
	}
	resp3 := <-rc
	if resp3.StatusCode != 200 {
		t.Error(unexpectErr(resp3.StatusCode, 200))
	}
}

func TestPoolGetBadURLs(t *testing.T) {
	p := NewPool(3)
	_, err := p.Get(badURLs)
	if err == nil {
		t.Error(errors.New("Should return an error."))
	}
}
