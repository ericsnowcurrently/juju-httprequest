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

type endpoint struct {
	Path string
}

func (ep endpoint) String() string {
	return path.Clean(path.Join("/", ep.Path))
}

func (ep endpoint) Name() string {
	return path.Base(ep.Path)
}

func (ep endpoint) Basic() Endpoint {
	return Endpoint{
		endpoint: endpoint{
			Path: ep.Path,
		},
	}
}

func (ep endpoint) resolve(pth ...string) endpoint {
	return endpoint{
		Path: path.Join(append([]string{ep.Path}, pth...)),
	}
}

func (ep Endpoint) request(method string, args interface{}) (*Request, error) {
	req := &Request{
		Method: method,
		Path:   c.Path,
	}

	if args != nil {
		var err error
		req.Args, err = normalizeArgs(args)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

func normalizeArgs(args interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(args)
	if err != nil {
		return nil, errgo.Notef(err, "could not normalize args")
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, errgo.Notef(err, "could not normalize args")
	}

	return result, nil
}

type Endpoint struct {
	endpoint
}

func (ep Endpoint) Resolve(path ...string) Endpoint {
	return ep.resolve(path...).Basic()
}

func (ep Endpoint) NewRequest(method string, args interface{}) (*Request, error) {
	return ep.request(method, args)
}

type Catagory struct {
	endpoint
}

func (c Category) Sub(name string) (*Category, error) {
	if name != path.Base(name) {
		return nil, errgo.Newf("nested names not supported (got %q)", name)
	}

	var sub Category
	sub.Path = path.Join(c.Path, name)
	return &sub, nil
}

func (c Category) Collection(name string) (*Collection, error) {
	if name != path.Base(name) {
		return nil, errgo.Newf("nested names not supported (got %q)", name)
	}

	var coll Collection
	coll.Path = path.Join(c.Path, name)
	return &coll, nil
}

type Collection struct {
	endpoint
}

func (c Collection) Item(id string) (*Item, error) {
	if id != path.Base(id) {
		return nil, errgo.Newf("nested IDs not supported (got %q)", id)
	}

	var item Item
	item.Path = path.Join(c.Path, name)
	return &item, nil
}

type ListArgs struct {
	Offset uint        `json:",omitempty"`
	Limit  uint        `json:",omitempty"`
	Filter interface{} `json:",omitempty"`
}

func (c Collection) List(args ListArgs) (*Request, error) {
	return c.request("GET", args)
}

func (c Collection) Create(args map[string]interface{}) (*Request, error) {
	return c.request("PUT", args)
}

type Item struct {
	endpoint
}

func (item Item) Info() (*Request, error) {
	return c.request("GET", nil)
}

//type ComplexItem struct {
//    Category
//}

//===========================

type Endpoint interface {
	Path() string
}

type Catagory interface {
	Endpoint

	Sub(name string) (Category, error)

	Collection(name string) (Collection, error)
}

type ListArgs struct {
	Offset uint

	Limit uint

	Filter interface{}
}

type Collection interface {
	Endpoint

	Item(id string) (Item, error)

	List(args ListArgs) (*Request, error)

	Create(args map[string]interface{}) (*Request, error)

	//List(results interface{}) error
	//Create(args interface{}) (Item, error)
}

type Item interface {
	Endpoint

	Info() (*Request, error)

	//Info(result interface{}) error
}

type DataItem interface {
	Item

	Upload(reader io.ReadSeeker, args, resp interface{}) error

	Download() (io.ReadCloser, error)
}
