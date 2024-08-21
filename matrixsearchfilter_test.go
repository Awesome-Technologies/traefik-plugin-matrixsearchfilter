package plugin_matrixsearchfilter

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

const exampleBody = `{
    "limited": false,
    "results": [
        {
            "user_id": "@abc:example.com",
            "display_name": "ABC",
            "avatar_url": null
        },
        {
            "user_id": "@efg_+:foo.example.com.bar",
            "display_name": "E FG",
            "avatar_url": "mxc://foo.example.com.bar/lNQvWxOnxiRINfNkcGA"
        },
        {
            "user_id": "@hij:bar.foo",
            "display_name": "HIJ"
        }
    ]
}`

func TestServeHTTP(t *testing.T) {
	tests := []struct {
		desc            string
		contentType     string
		contentEncoding string
		userIDRegex     string
		lastModified    bool
		resBody         string
		expResBody      string
		expLastModified bool
	}{
		{
			desc:        "should remove foo.example.com.bar and bar.foo",
			userIDRegex: "^@[a-z0-9\\._=\\-\\/\\+]+:example\\.com$",
			contentType: "application/json",
			resBody:     exampleBody,
			expResBody:  `{"limited":false,"results":[{"display_name":"ABC","user_id":"@abc:example.com"}]}`,
		},
		{
			desc:        "should not replace anything if content type is not JSON",
			userIDRegex: "^@[a-z0-9\\._=\\-\\/\\+]+:example\\.com$",
			contentType: "text",
			resBody:     "foo is the new bar",
			expResBody:  "foo is the new bar",
		},
		{
			desc:            "should not replace anything if content encoding is not identity or empty",
			userIDRegex:     "^@[a-z0-9\\._=\\-\\/\\+]+:example\\.com$",
			contentType:     "application/json",
			contentEncoding: "gzip",
			resBody:         "foo is the new bar",
			expResBody:      "foo is the new bar",
		},
		{
			desc:            "should remove foo.example.com.bar and bar.foo if content encoding is identity",
			userIDRegex:     "^@[a-z0-9\\._=\\-\\/\\+]+:example\\.com$",
			contentType:     "application/json",
			contentEncoding: "identity",
			resBody:         exampleBody,
			expResBody:      `{"limited":false,"results":[{"display_name":"ABC","user_id":"@abc:example.com"}]}`,
		},
		{
			desc:            "should not remove the last modified header",
			userIDRegex:     "^@[a-z0-9\\._=\\-\\/\\+]+:example\\.com$",
			contentType:     "application/json",
			contentEncoding: "identity",
			lastModified:    true,
			resBody:         exampleBody,
			expResBody:      `{"limited":false,"results":[{"display_name":"ABC","user_id":"@abc:example.com"}]}`,
			expLastModified: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				LastModified: test.lastModified,
				UserIDRegex:  test.userIDRegex,
			}

			next := func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Set("Content-Encoding", test.contentEncoding)
				rw.Header().Set("Last-Modified", "Thu, 02 Jun 2016 06:01:08 GMT")
				rw.Header().Set("Content-Length", strconv.Itoa(len(test.resBody)))
				rw.WriteHeader(http.StatusOK)

				req.Header.Set("Content-Type", test.contentType)

				_, _ = fmt.Fprintf(rw, test.resBody)
			}

			matrixSearchFilter, err := New(context.Background(), http.HandlerFunc(next), config, "matrixSearchFilter")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/_matrix/client/v3/user_directory/search", nil)

			matrixSearchFilter.ServeHTTP(recorder, req)

			if _, exists := recorder.Result().Header["Last-Modified"]; exists != test.expLastModified {
				t.Errorf("got last-modified header %v, want %v", exists, test.expLastModified)
			}

			if _, exists := recorder.Result().Header["Content-Length"]; exists {
				t.Error("The Content-Length Header must be deleted")
			}

			if !bytes.Equal([]byte(test.expResBody), recorder.Body.Bytes()) {
				t.Errorf("got body %q, want %q", recorder.Body.Bytes(), test.expResBody)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		desc        string
		userIDRegex string
		expErr      bool
	}{
		{
			desc:        "should return no error",
			userIDRegex: "^@[a-z0-9\\._=\\-\\/\\+]+:example\\.com$",
			expErr:      false,
		},
		{
			desc:        "should return an error",
			userIDRegex: "^@[a-z0-9\\._=\\-\\/\\++:example\\.com$",
			expErr:      true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				UserIDRegex: test.userIDRegex,
			}

			_, err := New(context.Background(), nil, config, "matrixSearchFilter")
			if test.expErr && err == nil {
				t.Fatal("expected error on bad regexp format")
			}
		})
	}
}
