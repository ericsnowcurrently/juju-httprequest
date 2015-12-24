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

type ConnInfo struct {
	Host string
	User string
}

func (ci ConnInfo) URL() *url.URL {
	var user *url.Userinfo
	if conn.User != "" {
		user = url.User(conn.User)
	}
	return &url.URL{
		Scheme: "https",
		Host:   ci.Host,
		User:   user,
	}
}

type Connection interface {
	Call(args, result interface{}) error
}
