// Copyright 2016 Jo Chasinga. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package requests is a minimal, atomic and expressive way of making HTTP requests.
It is inspired partly by the HTTP request libraries in other dynamic languages
like Python and Javascript. It is safe for all rodents, not just Gophers.

Requests is built as a convenient, expressive API around Go's standard http package.
With special Request and Response types to help facilitate and streamline RESTful tasks.

To send a common GET request just like you'd do with `http.Get`.

        import (
        	"github.com/jochasinga/requests"
        )

        func main() {
        	resp, err := requests.Get("http://httpbin.org/get")
        	fmt.Println(resp.StatusCode)  // 200
        }

To send additional data, such as a query parameter, basic authorization header,
or content type, just pass in functional options:

        addFoo := func(r *Request) {
                r.Params.Add("foo", "bar")
        }
        addAuth := func(r *Request) {
                r.SetBasicAuth("user", "pass")
        }
        addMime := func(r *Request) {
                r.Header.Add("content-type", "application/json")
        }
        resp, err := requests.Get("http://httpbin.org/get", addFoo, addAuth, addMime)

        // Or everything goes into one functional option
        opts := func(r *Request) {
                r.Params.Add("foo", "bar")
                r.SetBasicAuth("user", "pass")
                r.Header.Add("content-type", "application/json")
        }
        resp, err := requests.Get("http://httpbin.org/get", opts)

The data can be a map or struct (anything JSON-marshalable).

        data1 := map[string][]string{"foo": []string{"bar", "baz"}}
        data2 := struct {
                Foo []string `json:"foo"`
        }{[]string{"bar", "baz"}}

        data := map[string][]interface{}{
		"combined": {data1, data2},
	}

        res, err := requests.Post("http://httpbin.org/post", "application/json", data)

You can asynchronously wait on a GET response with `GetAsync`.

        timeout := time.Duration(1) * time.Second
        resChan, _ := requests.GetAsync("http://httpbin.org/get", nil, nil, timeout)

        // Do some other things

        res := <-resChan
        fmt.Println(res.StatusCode)  // 200

The response returned has the type *requests.Response which embeds *http.Response type
to provide more buffer-like methods such as Len(), String(), Bytes(), and JSON().

        // Len() returns the body's length.
        var len int = res.Len()

        // String() returns the body as a string.
        var text string = res.String()

        // Bytes() returns the body as bytes.
        var content []byte = res.Bytes()

        // JSON(), like Bytes() but returns an empty `[]byte` unless `Content-Type`
        // is set to `application/json` in the response's header.
        var jsn []byte = res.JSON()

These special methods use bytes.Buffer under the hood, thus unread portions of data
are returned. Make sure not to read from the response's body beforehand.

Requests is an ongoing project. Any contribution is whole-heartedly welcomed.

*/
package requests
