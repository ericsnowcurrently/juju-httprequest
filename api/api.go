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

type Info struct {
	RootPath     string
	ExtraHeaders http.Header
    Renderer     ArgsRenderer
    ArgsAsBody bool
	Formatter    Formatter
}

type Client interface {
	URL(path ...string) (*url.URL, error)

	Request(method string, path ...string) (*http.Request, error)

	Caller(method string, path ...string) (Caller, error)
}

type Caller interface {
	//Request(
}

type API interface {
	Endpoint
}

//-------------------------------

type Caller interface {

    Call(params, resp interface{}) error

    CallURL(url string, params, resp interface{}) error
}

func Download(client Caller, id string) (io.ReadCloser, error) {
	var resp *http.Response
    params := struct{

    }
	err := client.Call(
		&downloadParams{
			Body: params.BackupsDownloadArgs{
				ID: id,
			},
		},
		&resp,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return resp.Body, nil
}

type downloadParams struct {
	httprequest.Route `httprequest:"GET /backups"`
	Body              params.BackupsDownloadArgs `httprequest:",body"`
}

// Download returns an io.ReadCloser for the given backup id.
func (c *Client) Download(id string) (io.ReadCloser, error) {
	// Send the request.
	var resp *http.Response
	err := c.client.Call(
		&downloadParams{
			Body: params.BackupsDownloadArgs{
				ID: id,
			},
		},
		&resp,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return resp.Body, nil
}
