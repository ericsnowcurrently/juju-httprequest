// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"gopkg.in/errgo.v1"
)

type ArgsRenderer interface {
    Query(req Request) (url.Values, error)
	Body(req Request) (io.Reader, string, error)
}

//type Rendered struct {
//	Query    url.Values
//	Body     io.Reader
//	MimeType string
//}

type argsRenderer struct {
    argsAsBody func(Request) (io.Reader, string, error)
}

func (r argsRenderer) Render(req Request) (*Rendered, error) {
}

func (r argsRenderer) Render(req Request) (*Rendered, error) {
	if len(req.Args) == 0 {
		return &Rendered{Body: req.Data}, nil
	}

    if r.argsAsBody == nil {
        return r.argsAsQuery(req)
    }

    r.RenderBody(req)
}

type ArgsRenderer struct {
}

func (r ArgsRenderer) Render(req Request) (*Rendered, error) {

func (r ArgsRenderer) argsAsQuery(req Request) (*Rendered, error) {
	query, err := request2query(req)
	if err != nil {
		return nil, err
	}

	data := req.Data
	if data != nil {
		data = ioutil.NopCloser(data)
	}

	rendered := &Rendered{
		Query:    query,
		Body:     data,
		MimeType: "",
	}
	return rendered, nil
}

type BodyRenderer struct{
    Formatter
}

func (r BodyRenderer) Render(req Request) (*Rendered, error) {
	if len(req.Args) == 0 {
		return &Rendered{Body: req.Data}, nil
	}

	query, err := query(req)
	if err != nil {
		return nil, err
	}
	encoded := query.Encode()

	// Render the encoded data into the body.
	renderBody := r.queryOnly
	if req.Data != nil {
		renderBody = r.multipart
	}
	data, mimetype, err := renderBody(encoded, req.Data)
	if err != nil {
		return nil, err
	}

	rendered := &Rendered{
		// The query is rendered in the body.
		Query:    nil,
		Body:     data,
		MimeType: mimetype,
	}
	return rendered, nil
}

func (r BodyRenderer) queryOnly(query string, data io.Reader) (io.Reader, string, error) {
	buf := bytes.NewBufferString(encoded)
	mimetype := "application/x-www-form-urlencoded"
	return ioutil.NopCloser(buf), mimetype, nil
}

func (r BodyRenderer) multipart(query string, data io.Reader) (io.Reader, string, error) {
	//....
}

func request2query(req Request) (url.Values, error) {
    query := make(url.Values)
	for k, v := range req.Args {
		// TODO(ericsnow) Call Encode on the value?
		valStr := fmt.Sprintf("%s", v)
		query.Set(k, valStr)
	}
	return query, nil
}

type JSONRenderer struct{}
