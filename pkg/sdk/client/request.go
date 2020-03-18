/*
Copyright © 2020 Kevin Swiber <kswiber@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"

	"github.com/kevinswiber/postmanctl/pkg/sdk/resources"
)

// RequestError represents an error from the Postman API.
type RequestError struct {
	StatusCode int
	Name       string
	Message    string
}

// NewRequestError creates a new RequestError for Postman API responses.
func NewRequestError(code int, name string, message string) *RequestError {
	return &RequestError{
		StatusCode: code,
		Name:       name,
		Message:    message,
	}
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("status code: %d, name: %s, message: %s", e.StatusCode,
		e.Name, e.Message)
}

// Request holds state for a Postman API request.
type Request struct {
	ctx      context.Context
	c        *APIClient
	method   string
	resource string
	output   interface{}
	headers  http.Header
}

// NewRequest initializes a Postman API Request.
func NewRequest(c *APIClient) *Request {
	return NewRequestWithContext(context.Background(), c)
}

// NewRequestWithContext intiializes a Postman API Request with a
// given context.
func NewRequestWithContext(ctx context.Context, c *APIClient) *Request {
	r := &Request{
		ctx: ctx,
		c:   c,
	}

	if r.headers == nil {
		r.headers = http.Header{}
	}
	r.headers.Add("X-API-Key", c.APIKey)

	return r
}

// Method sets the HTTP method of the request.
func (r *Request) Method(m string) *Request {
	r.method = m
	return r
}

// Get sets the HTTP method to GET
func (r *Request) Get() *Request {
	r.method = "GET"
	return r
}

// Resource sets the path of the HTTP request.
func (r *Request) Resource(resource ...string) *Request {
	r.resource = path.Join(resource...)
	return r
}

// As sets a destination resource for the output response
func (r *Request) As(o interface{}) *Request {
	r.output = o
	return r
}

// URL returns a complete URL for the current request.
func (r *Request) URL() *url.URL {
	finalURL := &url.URL{}
	if r.c.base != nil {
		*finalURL = *r.c.base
	}
	finalURL.Path = r.resource

	return finalURL
}

// Do executes the HTTP request.
func (r *Request) Do() (*http.Response, error) {
	url := r.URL().String()
	req, err := http.NewRequestWithContext(r.ctx, r.method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header = r.headers
	client := r.c.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return resp, err
		}

		var e resources.ErrorResponse
		json.Unmarshal(body, &e)
		errorMessage := NewRequestError(resp.StatusCode, e.Error.Name, e.Error.Message)
		return nil, errorMessage
	}

	if r.output != nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return resp, err
		}

		json.Unmarshal(body, &r.output)
	}

	return resp, nil
}
