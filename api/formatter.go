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

type Formatter interface {
	//Encoding() []string
	Encode(args interface{}) ([]byte, error)
	Decode(data []byte, result interface{}) error
}

type JSON struct {
}
