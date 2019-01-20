// TODO: Break tests into multiple files.
package requests

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jochasinga/gtime"
	"github.com/jochasinga/relay"
)

func unexpectErr(result, expected interface{}) error {
	err := fmt.Errorf("[Expect] >> %v <<\n[Result] >> %v <<\n", expected, result)
	return err
}

var (
	// Functional options used in the tests.
	fn1 = func(r *Request) {
		r.Header.Set("content-type", "application/json")
	}
	fn2 = func(r *Request) {
		r.Timeout = time.Duration(3) * time.Second
	}
	fn3 = func(r *Request) {
		r.SetBasicAuth("user", "pass")
	}
	fn4 = func(r *Request) {
		r.Params.Add("foo", "bar")
	}
	fn5 = func(r *Request) {
		r.Params.Add("name", "Ava")
	}

	// Handler functions used in the test servers.
	helloHandler = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello world!")
	}
	jsonHandler = func(w http.ResponseWriter, r *http.Request) {
		data := []byte(`{"foo": ["bar", "baz"]}`)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
	jsonWithTypeParamHandler = func(w http.ResponseWriter, r *http.Request) {
		data := []byte(`{"foo": ["bar", "baz"]}`)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(data)
	}
	htmlHandler = func(w http.ResponseWriter, r *http.Request) {
		html := "<html><body><h1>Blanca!</h1></body></html>"
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, html)
	}
	multTypeHandler = func(w http.ResponseWriter, r *http.Request) {
		data := []byte(`{"foo": ["bar", "baz"]}`)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(data)
	}
	contentTypeHandler = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Header.Get("Content-Type"))
	}
	fooHandler = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.FormValue("foo"))
	}
	nameHandler = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.FormValue("name"))
	}
	basicAuthHandler = func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok {
			log.Panicln("Error getting Basic Auth.")
		}
		io.WriteString(w, user+" : "+password)
	}
	notFoundHandler = func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}
	echoPostHandler = func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		w.Header().Set("content-type", "application/json")
		w.Write(body)
	}
	optsHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("allow", "HEAD,GET,PUT,DELETE,OPTIONS")
		w.WriteHeader(200)
	}
	// Test tables and array of functional options to use in tests.
	optList = [...][]func(*Request){
		{fn1},
		{fn1, fn2},
		{fn1, fn2, fn3},
		{fn1, fn2, fn3, fn4},
		{fn1, fn2, fn3, fn4, fn5},
	}
	headFuncTestTable = []struct {
		fn       func(*Request)
		handler  func(http.ResponseWriter, *http.Request)
		expected int
	}{
		{fn1, notFoundHandler, 404},
		{fn3, notFoundHandler, 404},
		{fn4, notFoundHandler, 404},
		{fn5, notFoundHandler, 404},
	}
	getFuncTestTable = []struct {
		fn       func(*Request)
		handler  func(http.ResponseWriter, *http.Request)
		expected string
	}{
		{fn1, contentTypeHandler, "application/json"},
		{fn3, basicAuthHandler, "user : pass"},
		{fn4, fooHandler, "bar"},
		{fn5, nameHandler, "Ava"},
	}
	postFuncTestTable = []struct {
		bodyType string
		body     io.Reader
		opts     []func(*Request)
	}{
		{
			"application/json",
			bytes.NewBufferString(`{"foo":"bar"}`),
			[]func(*Request){fn1, fn2, fn3, fn4, fn5},
		},
		{
			"text/xml",
			strings.NewReader(`<foo>bar</foo>`),
			[]func(*Request){fn1, fn2, fn3, fn4, fn5},
		},
	}

	// Data for testing PostJSON
	bodyMap    = map[string][]string{"foo": []string{"bar", "baz"}}
	bodyStruct = struct {
		Foo []string `json:"foo"`
	}{[]string{"bar", "baz"}}
	bodyHybridMap = map[string][]interface{}{
		"duplica": {bodyMap, bodyStruct},
	}
	postJSONArgs = [...]interface{}{
		bodyMap, bodyStruct, bodyHybridMap,
	}

	// For testing timeouts
	getFuncSyncTestTable = []struct {
		delay    int
		expected int
	}{
		{1, 2},
		{2, 4},
		{3, 6},
		{4, 8},
	}
	getFuncTimeoutTestTable = []struct {
		delay    int
		timeout  float64
		expected float64
	}{
		{1, 0.5, 0.5},
		{2, 0.5, 0.5},
		{2, 1.0, 1.0},
		{3, 1.0, 1.0},
	}
	jsonFuncTestTable = []struct {
		fn       func(http.ResponseWriter, *http.Request)
		expected []byte
	}{
		{jsonHandler, []byte(`{"foo": ["bar", "baz"]}`)},
		{jsonWithTypeParamHandler, []byte(`{"foo": ["bar", "baz"]}`)},
		{multTypeHandler, []byte(`{"foo": ["bar", "baz"]}`)},
		{htmlHandler, []byte{}},
	}
	badURLs = []string{
		"://maggot.#&",
		"crap://bs.com",
		"htp://f#as3",
	}
)

// Test that the returned type is always *Response.
func TestHeadResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, opts := range optList {
		resp, err := Head(ts.URL, opts...)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

// Test that the request has the appropriate options.
func TestHeadResponseOptions(t *testing.T) {
	for _, tt := range headFuncTestTable {
		ts := httptest.NewServer(http.HandlerFunc(tt.handler))
		defer ts.Close()
		resp, err := Head(ts.URL, tt.fn)
		if err != nil {
			t.Error(err)
		}
		if resp.StatusCode != tt.expected {
			t.Error(err)
		}
	}
}

func TestHeadWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Head(url)
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

// Test that the returned type is always *Response.
func TestGetResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, opts := range optList {
		resp, err := Get(ts.URL, opts...)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

// Test that the request has the appropriate options.
func TestGetResponseOptions(t *testing.T) {
	for _, tt := range getFuncTestTable {
		ts := httptest.NewServer(http.HandlerFunc(tt.handler))
		defer ts.Close()
		resp, err := Get(ts.URL, tt.fn)
		if err != nil {
			t.Error(err)
		}
		if resp.String() != tt.expected {
			t.Error(err)
		}
	}
}

// Test that transport's attribute can be set on request
func TestGetWithTransport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	opt := func(r *Request) {
		r.TLSClientConfig = &tls.Config{RootCAs: x509.NewCertPool()}
		r.DisableCompression = true
	}
	resp, err := Get(ts.URL, opt)
	if err != nil {
		t.Error(err)
	}
	if resp.String() != "Hello world!" {
		t.Error(err)
	}
}

func paramsHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	for key := range values {
		fmt.Fprintf(w, "%s", key)
	}
}

// Get should favor query strings in the URL if provided instead of request.Params
func TestGetWithQueryString(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(paramsHandler))
	defer ts.Close()
	resp, err := Get(ts.URL + "?key1=hello&key2=world")
	if err != nil {
		t.Error(err)
	}
	if resp.String() != "key1key2" {
		t.Error("Parameters from query string were ignored")
	}
}

// TestGetWithParamsSet tests a situation when the Params is set.
func TestGetWithParamsSet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(paramsHandler))
	defer ts.Close()
	resp, err := Get(ts.URL, func(r *Request) {
		r.Params.Add("foo", "bar")
	})
	if err != nil {
		t.Error(err)
	}
	if resp.String() != "foo" {
		t.Errorf("Expect `foo`, got `%q`", resp.String())
	}
}

// TestGetWithQueryStringAndParamsSet tests a case when both the query string
// is provided in the URL and Params is set. Params should take precadence over
// the query string in this case.
func TestGetWithQueryStringAndParamsSet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(paramsHandler))
	defer ts.Close()

	resp, err := Get(ts.URL+"?hello=world", func(r *Request) {
		r.Params.Add("foo", "bar")
	})
	if err != nil {
		t.Error(err)
	}
	if resp.String() != "foo" {
		t.Errorf("Expect `foo`, got `%q`", resp.String())
	}
}

// Get should wait for the response and return
func TestGetResponseTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, tt := range getFuncSyncTestTable {
		delay := time.Duration(tt.delay) * time.Second
		expected := time.Duration(tt.expected) * time.Second
		p := relay.NewProxy(delay, ts)
		start := time.Now()
		_, _ = Get(p.URL)
		elapsed := time.Since(start)
		if elapsed <= expected {
			t.Error(unexpectErr(elapsed, expected))
		}
	}
}

// Get should wait fo the response until timed out.
func TestGetResponseOnTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, tt := range getFuncTimeoutTestTable {
		delay := time.Duration(tt.delay) * time.Second
		p := relay.NewProxy(delay, ts)
		start := time.Now()
		_, err := Get(p.URL, func(r *Request) {
			r.Timeout = gtime.Ftos(tt.timeout)
		})
		if err == nil {
			t.Error(err)
		}
		elapsed := time.Since(start).Seconds()
		deviation := gtime.FloatTime.Seconds()
		if !(elapsed >= tt.expected-deviation || elapsed <= tt.expected+deviation) {
			t.Error(unexpectErr(elapsed, tt.expected))
		}
	}
}

func TestGetWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Get(url)
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

func TestGetAsyncResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	rc, err := GetAsync(ts.URL)
	if err != nil {
		t.Error(err)
	}
	expectedType := reflect.TypeOf((<-chan *Response)(nil))
	resultType := reflect.TypeOf(rc)
	if resultType != expectedType {
		t.Error(unexpectErr(resultType, expectedType))
	}
}

// GetAsync should return immediately.
func TestGetAsyncResponseTimes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	deviation := time.Duration(10) * time.Millisecond
	for _, tt := range getFuncSyncTestTable {
		delay := time.Duration(tt.delay) * time.Second
		expected := time.Duration(tt.expected)*time.Second + deviation
		p := relay.NewProxy(delay, ts)
		start := time.Now()
		_, _ = GetAsync(p.URL)
		elapsed := time.Since(start)
		if elapsed >= expected {
			t.Error(unexpectErr(elapsed, expected))
		}
	}
}

// GetAsync should return immediately.
func TestGetAsyncResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	delay := time.Duration(1) * time.Second
	p := relay.NewProxy(delay, ts)
	rc, err := GetAsync(p.URL)
	if err != nil {
		t.Error(err)
	}
	resp := <-rc
	if resp.Error != nil {
		t.Error(resp.Error)
	}
	result := resp.String()
	if result != "Hello world!" {
		t.Error(unexpectErr(result, "Hello world!"))
	}
}

// FIXME: Cannot loop through bad URLs for this test
// probably due to the goroutine.
func TestGetAsyncWithBadURL(t *testing.T) {
	_, err := GetAsync(":ebg:htwe.com")
	if err == nil {
		t.Error(errors.New("Should return an error."))
	}
}

// Test if the response's body has the right type.
func TestPostResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, tt := range postFuncTestTable {
		resp, err := Post(ts.URL, tt.bodyType, tt.body, tt.opts...)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

// Test if the response's body is being echoed back right.
func TestPostResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(echoPostHandler))
	defer ts.Close()
	for _, arg := range postJSONArgs {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		resp, err := Post(ts.URL, "application/json", buf)
		if err != nil {
			t.Error(err)
		}
		b := new(bytes.Buffer)
		err = json.NewEncoder(b).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		result := resp.String()
		expected := b.String()
		if result != expected {
			t.Error(unexpectErr(result, expected))
		}
	}
}

func TestPostWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Post(url, "", &bytes.Buffer{})
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

func TestPostJSONResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, arg := range postJSONArgs {
		resp, err := PostJSON(ts.URL, arg)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

// Test if the response's body is being echoed back right.
func TestPostJSONResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(echoPostHandler))
	defer ts.Close()
	for _, arg := range postJSONArgs {
		resp, err := PostJSON(ts.URL, arg)
		if err != nil {
			t.Error(err)
		}
		b := new(bytes.Buffer)
		err = json.NewEncoder(b).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		result := resp.JSON()
		expected := b.Bytes()
		if bytes.Compare(result, expected) != 0 {
			t.Error(unexpectErr(result, expected))
		}
	}
}

func TestPostAsyncResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	rc, err := PostAsync(ts.URL, "", &bytes.Buffer{})
	if err != nil {
		t.Error(err)
	}
	expectedType := reflect.TypeOf((<-chan *Response)(nil))
	resultType := reflect.TypeOf(rc)
	if resultType != expectedType {
		t.Error(unexpectErr(resultType, expectedType))
	}
}

// PostAsync should return immediately.
func TestPostAsyncResponseTimes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	deviation := time.Duration(10) * time.Millisecond
	for _, tt := range getFuncSyncTestTable {
		delay := time.Duration(tt.delay) * time.Second
		expected := time.Duration(tt.expected)*time.Second + deviation
		p := relay.NewProxy(delay, ts)
		start := time.Now()
		_, _ = PostAsync(p.URL, "", &bytes.Buffer{})
		elapsed := time.Since(start)
		if elapsed >= expected {
			t.Error(unexpectErr(elapsed, expected))
		}
	}
}

func TestPostAsyncResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	rc, err := PostAsync(ts.URL, "", &bytes.Buffer{})
	if err != nil {
		t.Error(err)
	}
	resp := <-rc
	if resp.Error != nil {
		t.Error(resp.Error)
	}
	result := resp.String()
	if result != "Hello world!" {
		t.Error(unexpectErr(result, "Hello world!"))
	}
}

// FIXME: Cannot loop through bad URLs for this test
// probably due to the goroutine.
func TestPostAsyncWithBadURL(t *testing.T) {
	_, err := PostAsync(":ebg:htwe.com", "", &bytes.Buffer{})
	if err == nil {
		t.Error(errors.New("Should return an error."))
	}
}

// Test if the response's body has the right type.
func TestPutResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, tt := range postFuncTestTable {
		resp, err := Put(ts.URL, tt.bodyType, tt.body, tt.opts...)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

// Test if the response's body is being echoed back right.
func TestPutResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(echoPostHandler))
	defer ts.Close()
	for _, arg := range postJSONArgs {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		resp, err := Put(ts.URL, "application/json", buf)
		if err != nil {
			t.Error(err)
		}
		b := new(bytes.Buffer)
		err = json.NewEncoder(b).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		result := resp.String()
		expected := b.String()
		if result != expected {
			t.Error(unexpectErr(result, expected))
		}
	}
}

func TestPutWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Put(url, "", &bytes.Buffer{})
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

func TestPatchResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, tt := range postFuncTestTable {
		resp, err := Patch(ts.URL, tt.bodyType, tt.body, tt.opts...)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

func TestPatchResponseBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(echoPostHandler))
	defer ts.Close()
	for _, arg := range postJSONArgs {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		resp, err := Put(ts.URL, "application/json", buf)
		if err != nil {
			t.Error(err)
		}
		b := new(bytes.Buffer)
		err = json.NewEncoder(b).Encode(arg)
		if err != nil {
			t.Error(err)
		}
		result := resp.String()
		expected := b.String()
		if result != expected {
			t.Error(unexpectErr(result, expected))
		}
	}
}

func TestPatchWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Patch(url, "", &bytes.Buffer{})
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

// Test that the returned type is always *Response.
func TestDeleteResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	for _, opts := range optList {
		resp, err := Delete(ts.URL, opts...)
		if err != nil {
			t.Error(err)
		}
		resultType := reflect.TypeOf(resp)
		expectedType := reflect.TypeOf(&Response{})
		if resultType != expectedType {
			t.Error(unexpectErr(resultType, expectedType))
		}
	}
}

// Test that the returned type is always *Response.
func TestDeleteResponseBody(t *testing.T) {
	for _, tt := range getFuncTestTable {
		ts := httptest.NewServer(http.HandlerFunc(tt.handler))
		defer ts.Close()
		resp, err := Delete(ts.URL, tt.fn)
		if err != nil {
			t.Error(err)
		}
		if resp.String() != tt.expected {
			t.Error(unexpectErr(resp.String(), tt.expected))
		}
	}
}

func TestDeleteWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Delete(url)
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

// Test that the returned type is always *Response.
func TestOptionsResponseType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(optsHandler))
	defer ts.Close()
	resp, err := Options(ts.URL)
	if err != nil {
		t.Error(err)
	}
	resultType := reflect.TypeOf(resp)
	expectedType := reflect.TypeOf(&Response{})
	if resultType != expectedType {
		t.Error(unexpectErr(resultType, expectedType))
	}
}

// Test that the returned type is always *Response.
func TestOptionsResponseCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(optsHandler))
	defer ts.Close()
	resp, err := Options(ts.URL)
	if err != nil {
		t.Error(err)
	}
	result := resp.StatusCode
	expected := 200
	if result != expected {
		t.Error(unexpectErr(result, expected))
	}
}

// Test that the returned type is always *Response.
func TestOptionsResponseHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(optsHandler))
	defer ts.Close()
	resp, err := Options(ts.URL)
	if err != nil {
		t.Error(err)
	}
	result := resp.Header.Get("allow")
	expected := "HEAD,GET,PUT,DELETE,OPTIONS"
	if result != expected {
		t.Error(unexpectErr(result, expected))
	}
}

func TestOptionsWithBadURLs(t *testing.T) {
	for _, url := range badURLs {
		_, err := Options(url)
		if err == nil {
			t.Error(errors.New("Should return an error."))
		}
	}
}

func TestResponseLen(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	resp, err := Get(ts.URL)
	if err != nil {
		t.Error(err)
	}
	length := resp.Len()
	expected := len("Hello world!")
	if length != len("Hello world!") {
		t.Error(unexpectErr(length, expected))
	}
}

func TestResponseAsBytes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	resp, err := Get(ts.URL)
	if err != nil {
		t.Error(err)
	}
	result := resp.Bytes()
	expected := []byte("Hello world!")
	if bytes.Compare(result, expected) != 0 {
		t.Error(unexpectErr(result, expected))
	}
}

func TestResponseAsJSON(t *testing.T) {
	for _, tt := range jsonFuncTestTable {
		ts := httptest.NewServer(http.HandlerFunc(tt.fn))
		defer ts.Close()
		resp, err := Get(ts.URL)
		if err != nil {
			t.Error(err)
		}
		result := resp.JSON()
		if bytes.Compare(result, tt.expected) != 0 {
			t.Error(unexpectErr(result, tt.expected))
		}
	}
}

func TestResponseAsString(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(helloHandler))
	defer ts.Close()
	resp, err := Get(ts.URL)
	if err != nil {
		t.Error(err)
	}
	result := resp.String()
	if result != "Hello world!" {
		t.Error(unexpectErr(result, "Hello world!"))
	}
}

func TestResponseContentType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(jsonWithTypeParamHandler))
	defer ts.Close()
	resp, err := Get(ts.URL)
	if err != nil {
		t.Error(err)
	}
	result, params, err := resp.ContentType()
	if err != nil {
		t.Error(err)
	}
	resultParam := params["charset"]
	expectedParam := "utf-8"
	if resultParam != expectedParam {
		t.Error(unexpectErr(resultParam, expectedParam))
	}
	expected, _, err := mime.ParseMediaType("application/json")
	if err != nil {
		t.Error(err)
	}
	if result != expected {
		t.Error(unexpectErr(result, expected))
	}
}

func TestResponseWithoutContentType(t *testing.T) {
	resp := &Response{
		Response: &http.Response{
			Header: http.Header{},
		},
	}
	_, _, err := resp.ContentType()
	if err == nil {
		t.Error(err)
	}
}

/*********************** TRANSPORT TEST ***************************/
