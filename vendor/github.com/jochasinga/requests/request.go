package requests

import (
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

// HTTPRequest is for future uses.
type HTTPRequest interface {
	AddCookie(*http.Cookie)
	BasicAuth() (string, string, bool)
	Cookie(string) (*http.Cookie, error)
	Cookies() []*http.Cookie
	FormFile(string) (multipart.File, *multipart.FileHeader, error)
	FormValue(string) string
	MultipartReader() (*multipart.Reader, error)
	ParseForm() error
	ParseMultipartForm(int64) error
	PostFormValue(string) string
	ProtoAtLeast(int, int) bool
	Referer() string
	SetBasicAuth(string, string)
	UserAgent() string
	Write(io.Writer) error
	WriteProxy(io.Writer) error
}

// Request is a hybrid descendant of http.Request and http.Client.
// It is used as an argument to the functional options.
type Request struct {

	// "Inherite" both http.Request and http.Client
	*http.Request
	*http.Client
	*http.Transport

	// Params is an alias used to add query parameters to
	// the request through functional options.
	// Values in Params have higher precedence over the
	// query string in the initial URL.
	Params url.Values
}
