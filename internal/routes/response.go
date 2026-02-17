package routes

import "net/http"

// ResponseMeta contains HTTP-level metadata that handlers can return
// to control redirects, cookies, headers, etc.
type ResponseMeta struct {
	Redirect *RedirectInfo
	Cookies  []*http.Cookie
	Headers  map[string]string
}

// RedirectInfo specifies a redirect response
type RedirectInfo struct {
	StatusCode int    // e.g., http.StatusSeeOther (303)
	Location   string // redirect URL
}

// NewResponseMeta creates an empty ResponseMeta
func NewResponseMeta() *ResponseMeta {
	return &ResponseMeta{
		Headers: make(map[string]string),
	}
}

// WithRedirect sets redirect information
func (m *ResponseMeta) WithRedirect(statusCode int, location string) *ResponseMeta {
	m.Redirect = &RedirectInfo{
		StatusCode: statusCode,
		Location:   location,
	}
	return m
}

// WithCookie adds a cookie to be set
func (m *ResponseMeta) WithCookie(cookie *http.Cookie) *ResponseMeta {
	m.Cookies = append(m.Cookies, cookie)
	return m
}

// WithHeader adds a header to be set
func (m *ResponseMeta) WithHeader(key, value string) *ResponseMeta {
	m.Headers[key] = value
	return m
}
