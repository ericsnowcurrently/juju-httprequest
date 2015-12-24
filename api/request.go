// Copyright 2015 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package api

import (
	"net/http"
	"net/url"
	"path"
)

type Request struct {
	Method string
	Path   string
	Args   map[string]interface{}
	Data   io.Reader
}

func (req Request) URL(conn ConnInfo, api Info) (*url.URL, error) {
	renderer := renderer(api)

	reqURL := conn.URL()

	reqURL.Path = path.Clean(path.Join("/", api.RootPath, req.Path))

	query, err := renderer.Query(req)
	if err != nil {
		return nil, err
	}
	reqURL.RawQuery = query.Encode()

	return reqURL, nil
}

func (req Request) HTTPRequest(conn ConnInfo, api Info) (*http.Request, error) {
	reqURL, err := req.URL(conn, api)
	if err != nil {
		return nil, err
	}

	body := req.Data
	contentType := ""
	if api.ArgsAsBody {
		data, mimetype, err := req.bodyWithArgs()
		if err != nil {
			return nil, err
		}
		body = data
		contextType = mimetype
	}

	raw, err := http.NewRequest(req.Method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}

	for k, vs := range api.ExtraHeaders {
		for _, v := range vs {
			raw.Header.Add(k, v)
		}
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return raw, nil
}

func (req Request) bodyWithArgs() (io.Reader, string, error) {
	renderer := renderer(api)

	body, mimetype, err := renderer.Body(req)
	if err != nil {
		return nil, err
	}
	//....
}

func renderer(api Info) Renderer {
	renderer := api.ArgsRenderer
	if renderer == nil {
		renderer = &BasicRenderer{}
	}
	return renderer
}
