requests
========
[![GoDoc](https://godoc.org/github.com/jochasinga/requests?status.svg)](https://godoc.org/github.com/jochasinga/requests)   [![Build Status](https://travis-ci.org/jochasinga/requests.svg?branch=master)](https://travis-ci.org/jochasinga/requests)   [![Coverage Status](https://coveralls.io/repos/github/jochasinga/requests/badge.svg?branch=master)](https://coveralls.io/github/jochasinga/requests?branch=master)     [![Donate](https://img.shields.io/badge/donate-$1-yellow.svg)](https://www.paypal.me/jochasinga/1.00)

The simplest and functional HTTP requests in Go. 

Introduction
------------
Ever find yourself going back and forth between [net/http](https://golang.org/pkg/net/http/) and [io](https://golang.org/pkg/io/) docs and your code while making HTTP calls? **Requests** takes care of that by abstracting several types on both ends to just `Request` and `Response`--just the way it should be. [Find out how](#usage) or [jump right in to examples](#examples).

Purpose
-------
I need an HTTP request package that:
+ is atomic--All HTTP request configurations should be set, intuitively, on the request only.
+ wraps useful channels-ridden [asynchronous patterns](#asynchronous-apis).
+ has helper functions like [marshaling and posting JSON](#requestspostjson).
+ stays true to [net/http](https://golang.org/pkg/net/http/) APIs.
+ is idiomatic Go.

Usage
-----
the following are the core differences from the standard `net/http` package.

### Functional Options
requests employs functional options as optional parameters, this approach being
idiomatic, clean, and [makes a friendly, extensible API](http://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis).
This pattern is adopted after feedback from the Go community.

```go

jsontype := func(r *requests.Request) {
        r.Header.Add("content-type", "application/json")
}
res, err := requests.Get("http://example.com", jsontype)

```

### Embedded Standard Types
requests uses custom [Request](https://godoc.org/github.com/jochasinga/requests#Request)
and [Response](https://godoc.org/github.com/jochasinga/requests#Response) types
to embed standard [http.Request](https://golang.org/pkg/net/http/#Request), [http.Response](https://golang.org/pkg/net/http/#Response), and [http.Client](https://golang.org/pkg/net/http/#Client)
in order to insert helper methods, make configuring options **atomic**,
and [handle asynchronous errors](#handling-async-errors).

The principle is, the caller should be able to set all the configurations on the
"Request" instead of doing it on the client, transport, vice versa. For instance,
`Timeout` can be set on the `Request`.

```go

timeout := func(r *requests.Request) {

        // Set Timeout on *Request instead of *http.Client
        r.Timeout = time.Duration(5) * time.Second
}
res, err := requests.Get("http://example.com", timeout)
if err != nil {
        panic(err)
}

// Helper method
htmlStr := res.String()

```

Also, where `http.Transport` was normally needed to set control over
proxies, TLS configuration, keep-alives, compression, and other settings,
now everything is handled by `Request`.

```go

tlsConfig := func(r *requests.Request) {
        r.TLSClientConfig = &tls.Config{RootCAs: x509.NewCertPool()}
        r.DisableCompression = true
}
res, _ := requests.Get("http://example.com", tlsConfig)

```

See [Types and Methods](#types-and-methods) for more information.

### Asynchronous APIs
requests provides the following abstractions around sending HTTP requests in goroutines:
+ `requests.GetAsync`
+ `requests.PostAsync`
+ `requests.Pool`

All return a receive-only channel on which `*requests.Response` can be waited on.

```go

rc, err := requests.GetAsync("http://httpbin.org/get")
if err != nil {
        panic(err)
}
res := <-rc

// Handle connection errors.
if res.Error != nil {
        panic(res.Error)
}

// Helper method
content := res.Bytes()

```

See [Async](#async) and [Handling Async Errors](#handling-async-errors)
for more usage information.

Install
-------

```bash

go get github.com/jochasinga/requests

```

Testing
-------
requests uses Go standard `testing` package. Run this in the project's directory:

```bash

go test -v -cover

```

Examples
--------
### `requests.Get`
Sending a basic GET request is straightforward.

```go

res, err := requests.Get("http://httpbin.org/get")
if err != nil {
        panic(err)
}

fmt.Println(res.StatusCode)  // 200

```

To send additional data, such as a query parameter, or set basic authorization header
or content type, use functional options.

```go

// Add a query parameter.
addFoo := func(r *requests.Request) {
        r.Params.Add("foo", "bar")
}

// Set basic username and password.
setAuth := func(r *requests.Request) {
        r.SetBasicAuth("user", "pass")
}

// Set the Content-Type.
setMime := func(r *requests.Request) {
        r.Header.Add("content-type", "application/json")
}

// Pass as parameters to the function.
res, err := requests.Get("http://httpbin.org/get", addFoo, setAuth, setMime)

```

Or configure everything in one function.

```go

opts := func(r *requests.Request) {
        r.Params.Add("foo", "bar")
        r.SetBasicAuth("user", "pass")
        r.Header.Add("content-type", "application/json")
}
res, err := requests.Get("http://httpbin.org/get", opts)

```

### `requests.Post`
Send POST requests with specified `bodyType` and `body`.

```go

res, err := requests.Post("https://httpbin.org/post", "image/jpeg", &buf)

```

It also accepts variadic number of functional options:

```go

notimeout := func(r *requests.Request) {
        r.Timeout = 0
}
res, err := requests.Post("https://httpbin.org/post", "application/json", &buf, notimeout)

```

### `requests.PostJSON`
Encode your map or struct data as JSON and set `bodyType` to `application/json` implicitly.

```go

first := map[string][]string{
        "foo": []string{"bar", "baz"},
}
second := struct {
        Foo []string `json:"foo"`
}{[]string{"bar", "baz"}}

payload := map[string][]interface{}{
        "twins": {first, second}
}

res, err := requests.PostJSON("https://httpbin.org/post", payload)

```

### Other Verbs
`HEAD`, `PUT`, `PATCH`, `DELETE`, and `OPTIONS` are supported. See the [doc]("https://godoc.org/github.com/jochasinga/requests") for more info.

Async
-----
### `requests.GetAsync`
After parsing all the options, `GetAsync` spawns a goroutine to send a GET request and return `<-chan *Response` immediately on which `*Response` can be waited.

```go

timeout := func(r *requests.Request) {
        r.Timeout = time.Duration(5) * time.Second
}

rc, err := requests.GetAsync("http://golang.org", timeout)
if err != nil {
        panic(err)
}

// Do other things...

// Block and wait
res := <-rc

// Handle a "reject" with Error field.
if res.Error != nil {
	panic(res.Error)
}

fmt.Println(res.StatusCode)  // 200

```

`select` can be used to poll many channels asynchronously like normal.

```go

res1, _ := requests.GetAsync("http://google.com")
res2, _ := requests.GetAsync("http://facebook.com")
res3, _ := requests.GetAsync("http://docker.com")

for i := 0; i < 3; i++ {
        select {
    	case r1 := <-res1:
    		fmt.Println(r1.StatusCode)
    	case r2 := <-res2:
    		fmt.Println(r2.StatusCode)
    	case r3 := <-res3:
    		fmt.Println(r3.StatusCode)
    	}
}

```

[requests.Pool](#requestspool) is recommended for collecting concurrent responses from multiple requests.

### `requests.PostAsync`
An asynchronous counterpart of `requests.Post`.

```go

query := bytes.NewBufferString(`{
        "query" : {
            "term" : { "user" : "poco" }
        }
}`)

// Sending query to Elasticsearch server
rc, err := PostAsync("http://localhost:9200/users/_search", "application/json", query)
if err != nil {
        panic(err)
}
resp := <-rc
if resp.Error != nil {
        panic(resp.Error)
}
result := resp.JSON()

```

### `requests.Pool`
Contains a `Responses` field of type `chan *Response` with variable-sized buffer specified in the constructor. `Pool` is used to collect in-bound responses sent from numbers of goroutines corresponding
to the number of URLs provided in the slice.

```go

urls := []string{
        "http://golang.org",
        "http://google.com",
        "http://docker.com",
        "http://medium.com",
        "http://example.com",
        "http://httpbin.org/get",
        "https://en.wikipedia.org",
}

// Create a pool with the maximum buffer size.
p := requests.NewPool(10)
opts := func(r *requests.Request) {
        r.Header.Set("user-agent", "GoBot(http://example.org/"))
        r.Timeout = time.Duration(10) * time.Second
}

results, err := p.Get(urls, opts)

// An error is returned when an attempt to construct a
// request fails, probably from a malformed URL.
if err != nil {
        panic(err)
}
for res := range results {
        if res.Error != nil {
                panic(res.Error)
        }
        fmt.Println(res.StatusCode)
}

```

You may want to ignore errors from malformed URLs instead of handling each of them,
for instance, when crawling mass URLs.

To suppress the errors from being returned, either thrown away the error with `_` or
set the `IgnoreBadURL` field to `true`, which suppress all internal errors 
from crashing the pool:

```go

results, err := p.Get(urls, func(r *requests.Request) {
        r.IgnoreBadURL = true
})

```

`Pool.Responses` channel is closed internally when all the responses are sent.

Types and Methods
-----------------
### `requests.Request`
It has embedded types `*http.Request` and `*http.Client`, making it an atomic
type to pass into a functional option.
It also contains field `Params`, which has the type `url.Values`. Use this field
to add query parameters to your URL. Currently, parameters in `Params` will replace
all the existing query string in the URL.

```go

addParams := func(r *requests.Request) {
        r.Params = url.Values{
	        "name" : { "Ava", "Sanchez", "Poco" },
        }
}

// "q=cats" will be replaced by the new query string
res, err := requests.Get("https://httpbin.org/get?q=cats", addParams)

```

### `requests.Response`
It has embedded type `*http.Response` and provides extra byte-like helper methods
such as:
+ `Len() int`
+ `String() string`
+ `Bytes() []byte`
+ `JSON() []byte`

These methods will return an equivalent of `nil` for each return type if a
certain condition isn't met. For instance:

```go

res, _ := requests.Get("http://somecoolsite.io")
fmt.Println(res.JSON())

```

If the response from the server does not specify `Content-Type` as "application/json",
`res.JSON()` will return an empty bytes slice. It does not panic if the content type
is empty.

These methods close the response's body automatically.

Another helper method, `ContentType`, is used to get the media type in the
response's header, and can be used with the helper methods to determine the
type before reading the output.

```go

mime, _, err := res.ContentType()
if err != nil {
        panic(err)
}

switch mime {
case "application/json":
        fmt.Println(res.JSON())
case "text/html", "text/plain":
        fmt.Println(res.String())
default:
        fmt.Println(res.Bytes())
}

```

### Handling Async Errors
`requests.Response` also has an `Error` field which will contain any error
caused in the goroutine within `requests.GetAsync` and carries it downstream
for proper handling (Think `reject` in Promise but more
straightforward in Go-style).

```go

rc, err := requests.GetAsync("http://www.docker.io")

// This error is returned before the goroutine i.e. malformed URL.
if err != nil {
        panic(err)
}

res := <-rc

// This connection error is "attached" to the response.
if res.Error != nil {
	panic(res.Error)
}

fmt.Println(res.StatusCode)

```

`Response.Error` is default to `nil` when there is no error or when the response
is being retrieved from a synchronous function.

HTTP Test Servers
-----------------
Check out my other project [relay](https://github.com/jochasinga/relay),
useful test proxies and round-robin switchers for end-to-end HTTP tests.

Contribution
------------
Yes, please fork away.

Disclaimer
----------
To support my ends in NYC and help me push commits, please consider [![Donate](https://img.shields.io/badge/donate-$3-yellow.svg)](https://www.paypal.me/jochasinga/1.00) to fuel me with quality 🍵 or 🌟 this repo for spiritual octane.    
Reach me at [@jochasinga](http://twitter.com/jochasinga).
