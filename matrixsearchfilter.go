// Package plugin_matrixsearchfilter a plugin to rewrite response body of matrix user search.
package plugin_matrixsearchfilter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
)

// Config holds the plugin configuration.
type Config struct {
	LastModified bool   `json:"lastModified,omitempty"`
	UserIDRegex  string `json:"userIdRegex,omitempty"`
}

// CreateConfig creates and initializes the plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// MatrixUser holds a matrix user search result.
type MatrixUser struct {
	AvatarURL   string `json:"avatar_url,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	UserID      string `json:"user_id"`
}

// MatrixSearchResult returned in the HTTP response.
type MatrixSearchResult struct {
	Limited bool         `json:"limited"`
	Results []MatrixUser `json:"results"`
}

type matrixSearchFilter struct {
	name         string
	next         http.Handler
	UserIDRegex  *regexp.Regexp
	lastModified bool
}

// New creates and returns a new matrix search filter plugin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	regex, err := regexp.Compile(config.UserIDRegex)
	if err != nil {
		return nil, fmt.Errorf("error compiling regex %q: %w", config.UserIDRegex, err)
	}

	return &matrixSearchFilter{
		name:         name,
		next:         next,
		UserIDRegex:  regex,
		lastModified: config.LastModified,
	}, nil
}

func filterResponse(bodyBytes []byte, userIDRegex *regexp.Regexp) ([]byte, error) {
	var searchRes MatrixSearchResult

	err := json.Unmarshal(bodyBytes, &searchRes)
	if err != nil {
		log.Printf("unable to decode JSON body: %v", err)

		return nil, err
	}

	count := 0

	for _, user := range searchRes.Results {
		if userIDRegex.MatchString(user.UserID) {
			searchRes.Results[count] = user
			count++
		}
	}

	searchRes.Results = searchRes.Results[:count]

	bodyBytes, err = json.Marshal(searchRes)
	if err != nil {
		log.Printf("unable to encode JSON body: %v", err)

		return nil, err
	}

	return bodyBytes, nil
}

func (r *matrixSearchFilter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		lastModified:   r.lastModified,
		ResponseWriter: rw,
	}

	r.next.ServeHTTP(wrappedWriter, req)

	bodyBytes := wrappedWriter.buffer.Bytes()

	contentType := req.Header.Get("Content-Type")
	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")

	if req.Method != "POST" || req.URL.Path != "/_matrix/client/v3/user_directory/search" ||
		contentType != "application/json" || (contentEncoding != "" && contentEncoding != "identity") {
		if _, err := rw.Write(bodyBytes); err != nil {
			log.Printf("unable to write body: %v", err)
		}

		return
	}

	bodyBytes, err := filterResponse(bodyBytes, r.UserIDRegex)
	if err != nil {
		return
	}

	if _, err := rw.Write(bodyBytes); err != nil {
		log.Printf("unable to write rewrited body: %v", err)
	}
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool

	http.ResponseWriter
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if !r.lastModified {
		r.ResponseWriter.Header().Del("Last-Modified")
	}

	r.wroteHeader = true

	// Delegates the Content-Length Header creation to the final body write.
	r.ResponseWriter.Header().Del("Content-Length")

	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.buffer.Write(p)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.ResponseWriter)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
