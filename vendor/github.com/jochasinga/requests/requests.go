// Package requests provide useful and declarative methods for
// RESTful HTTP requests.
package requests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

func wrapRequest(method, urlStr string, body io.Reader, options []func(*Request)) (*Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	request := &Request{
		Request:   req,
		Client:    &http.Client{},
		Transport: &http.Transport{},
		Params:    url.Values{},
	}

	// Apply options in the parameters to request.
	for _, option := range options {
		option(request)
	}
	sURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// If request.Params is set, replace raw query with that.
	if len(request.Params) > 0 {
		sURL.RawQuery = request.Params.Encode()
	}

	req.URL = sURL

	// Parse query values into r.Form
	err = req.ParseForm()
	if err != nil {
		return nil, err
	}
	return request, nil
}

// Head sends a HTTP HEAD request to the provided url with the
// functional options to add query paramaters, headers, timeout, etc.
func Head(urlStr string, options ...func(*Request)) (*Response, error) {
	request, err := wrapRequest("HEAD", urlStr, nil, options)
	if err != nil {
		return nil, err
	}
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// Get sends a HTTP GET request to the provided url with the
// functional options to add query paramaters, headers, timeout, etc.
//
//     addMimeType := func(r *Request) {
//             r.Header.Add("content-type", "application/json")
//     }
//
//     resp, err := requests.Get("http://httpbin.org/get", addMimeType)
//     if err != nil {
//             panic(err)
//     }
//     fmt.Println(resp.StatusCode)
//
func Get(urlStr string, options ...func(*Request)) (*Response, error) {
	request, err := wrapRequest("GET", urlStr, nil, options)
	if err != nil {
		return nil, err
	}
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// GetAsync sends a HTTP GET request to the provided URL and
// returns a <-chan *http.Response immediately.
//
//     timeout := func(r *request.Request) {
//             r.Timeout = time.Duration(10) * time.Second
//     }
//     rc, err := requests.GetAsync("http://httpbin.org/get", timeout)
//     if err != nil {
//             panic(err)
//     }
//     resp := <-rc
//     if resp.Error != nil {
//             panic(resp.Error)
//     }
//     fmt.Println(resp.String())
//
func GetAsync(urlStr string, options ...func(*Request)) (<-chan *Response, error) {
	request, err := wrapRequest("GET", urlStr, nil, options)
	if err != nil {
		return nil, err
	}
	rc := make(chan *Response)
	go func() {
		resp, err := request.Client.Do(request.Request)
		// Wrap *http.Response with *Response
		response := &Response{}
		if err != nil {
			response.Error = err
			rc <- response
		}
		response.Response = resp
		rc <- response
		close(rc)
	}()
	return rc, nil
}

// Post sends a HTTP POST request to the provided URL, and
// encode the data according to the appropriate bodyType.
//
// redirect := func(r *requests.Request) {
//           r.CheckRedirect = redirectPolicyFunc
// }
// resp, err := requests.Post("https://httpbin.org/post", "image/png", &buf, redirect)
// if err != nil {
//         panic(err)
// }
// fmt.Println(resp.JSON())
//
func Post(urlStr, bodyType string, body io.Reader, options ...func(*Request)) (*Response, error) {
	request, err := wrapRequest("POST", urlStr, body, options)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", bodyType)
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// PostAsync sends a HTTP POST request to the provided URL, body type,
// and body, and returns a <-chan *http.Response immediately.
//
// redirect := func(r *requests.Request) {
//           r.CheckRedirect = redirectPolicyFunc
// }
// resp, err := requests.PostAsync("https://httpbin.org/post", "image/png", &buf, redirect)
// if err != nil {
//         panic(err)
// }
//
// resp := <-rc
// if resp.Error != nil {
//     panic(resp.Error)
// }
// fmt.Println(resp.String())
//
func PostAsync(urlStr, bodyType string, body io.Reader, options ...func(*Request)) (<-chan *Response, error) {
	request, err := wrapRequest("POST", urlStr, body, options)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", bodyType)
	rc := make(chan *Response)
	go func() {
		resp, err := request.Client.Do(request.Request)
		// Wrap *http.Response with *Response
		response := &Response{}
		if err != nil {
			response.Error = err
			rc <- response
		}
		response.Response = resp
		rc <- response
		close(rc)
	}()
	return rc, nil
}

// PostJSON aka UnsafePost! It marshals your data as JSON and set the bodyType
// to "application/json" automatically.
//
// redirect := func(r *requests.Request) {
//         r.CheckRedirect = redirectPolicyFunc
// }
//
// first := map[string][]string{"foo": []string{"bar", "baz"}}
// second := struct {Foo []string `json:"foo"`}{[]string{"bar", "baz"}}
// payload := map[string][]interface{}{"twins": {first, second}}
//
// resp, err := requests.PostJSON("https://httpbin.org/post", payload, redirect)
// if err != nil {
//         panic(err)
// }
// fmt.Println(resp.StatusCode)
//
func PostJSON(urlStr string, body interface{}, options ...func(*Request)) (*Response, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(body)
	if err != nil {
		return nil, err
	}
	request, err := wrapRequest("POST", urlStr, buf, options)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// Put sends HTTP PUT request to the provided URL.
func Put(urlStr, bodyType string, body io.Reader, options ...func(*Request)) (*Response, error) {
	request, err := wrapRequest("PUT", urlStr, body, options)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", bodyType)
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// Patch sends a HTTP PATCH request to the provided URL with optional body to modify data.
func Patch(urlStr, bodyType string, body io.Reader, options ...func(*Request)) (*Response, error) {
	request, err := wrapRequest("PATCH", urlStr, body, options)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", bodyType)
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// Delete sends a HTTP DELETE request to the provided URL.
func Delete(urlStr string, options ...func(*Request)) (*Response, error) {
	request, err := wrapRequest("DELETE", urlStr, nil, options)
	if err != nil {
		return nil, err
	}
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}

// Options sends a rarely-used HTTP OPTIONS request to the provided URL.
// Options only allows one parameter--the destination URL string.
func Options(urlStr string) (*Response, error) {
	request, err := wrapRequest("OPTIONS", urlStr, nil, []func(r *Request){})
	if err != nil {
		return nil, err
	}
	resp, err := request.Client.Do(request.Request)
	if err != nil {
		return nil, err
	}

	// Wrap *http.Response with *Response
	response := &Response{Response: resp}
	return response, nil
}
