// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtocol(t *testing.T) {
	r := ioutil.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","method":"Service2.Multiply","params":{"A":2, "B":4},"id":1}`))
	s := newRawCodecRequest(r)

	method, err := s.Method()
	assert.Nil(t, err)
	assert.Equal(t, method, "Service2.Multiply")

	arg := new(Service2Request)
	s.ReadRequest(arg)
	assert.Equal(t, arg, &Service2Request{A: 2, B: 4})

	reply := &Service2Response{Result: 8}
	w := new(bytes.Buffer)
	s.WriteResponse(w, reply)
	assert.Equal(t, w.String(), `{"jsonrpc":"2.0","result":{"Result":8},"id":1}`+"\n")
}

type Service2Request struct {
	A int
	B int
}

type Service2Response struct {
	Result int
}

const Service2DefaultResponse = 9999

type Service2 struct {
}

func (t *Service2) Multiply(r *http.Request, req *Service2Request, res *Service2Response) error {
	if req.A == 0 && req.B == 0 {
		// Sentinel value for test with no params.
		res.Result = Service2DefaultResponse
	} else {
		res.Result = req.A * req.B
	}
	return nil
}

func (t *Service2) ResponseError(r *http.Request, req *Service2Request, res *Service2Response) error {
	return ErrResponseError
}
