// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package jsonrpc2

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/terryding77/rpc"
)

var null = json.RawMessage([]byte("null"))

// Version means this is json-rpc 2.0 protocol
var Version = "2.0"

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

// serverRequest represents a JSON-RPC request received by the server.
type serverRequest struct {
	// JSON-RPC protocol.
	Version string `json:"jsonrpc"`

	// A String containing the name of the method to be invoked.
	Method string `json:"method"`

	// A Structured value to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`

	// The request id. MUST be a string, number or null.
	// Our implementation will not do type checking for id.
	// It will be copied as it is.
	ID *json.RawMessage `json:"id"`
}

// serverResponse represents a JSON-RPC response returned by the server.
type serverResponse struct {
	// JSON-RPC protocol.
	Version string `json:"jsonrpc"`

	// The Object that was returned by the invoked method. This must be null
	// in case there was an error invoking the method.
	// As per spec the member will be omitted if there was an error.
	Result interface{} `json:"result,omitempty"`

	// An Error object if there was an error invoking the method. It must be
	// null if there was no error.
	// As per spec the member will be omitted if there was no error.
	Error *Error `json:"error,omitempty"`

	// This must be the same id as the request it is responding to.
	ID *json.RawMessage `json:"id"`
}

// ----------------------------------------------------------------------------
// rawCodecRequest
// ----------------------------------------------------------------------------

// newCodecRequest returns a new CodecRequest.
func newRawCodecRequest(r io.ReadCloser) *rawCodecRequest {
	// Decode the request body and check if RPC method is valid.
	req := new(serverRequest)
	err := json.NewDecoder(r).Decode(req)
	if err != nil {
		err = &Error{
			Code:    E_PARSE,
			Message: err.Error(),
			Data:    req,
		}
	}
	if req.Version != Version {
		err = &Error{
			Code:    E_INVALID_REQ,
			Message: "jsonrpc must be " + Version,
			Data:    req,
		}
	}
	r.Close()
	return &rawCodecRequest{request: req, err: err}
}

// CodecRequest decodes and encodes a single request.
type rawCodecRequest struct {
	request *serverRequest
	err     error
}

// Method returns the RPC method for the current request.
//
// The method uses a dotted notation as in "Service.Method".
func (c *rawCodecRequest) Method() (string, error) {
	if c.err == nil {
		return c.request.Method, nil
	}
	return "", c.err
}

// ReadRequest fills the request object for the RPC method.
//
// ReadRequest parses request parameters in two supported forms in
// accordance with http://www.jsonrpc.org/specification#parameter_structures
//
// by-position: params MUST be an Array, containing the
// values in the Server expected order.
//
// by-name: params MUST be an Object, with member names
// that match the Server expected parameter names. The
// absence of expected names MAY result in an error being
// generated. The names MUST match exactly, including
// case, to the method's expected parameters.
func (c *rawCodecRequest) ReadRequest(args interface{}) error {
	if c.err == nil && c.request.Params != nil {
		// Note: if c.request.Params is nil it's not an error, it's an optional member.
		// JSON params structured object. Unmarshal to the args object.
		if err := json.Unmarshal(*c.request.Params, args); err != nil {
			// Clearly JSON params is not a structured object,
			// fallback and attempt an unmarshal with JSON params as
			// array value and RPC params is struct. Unmarshal into
			// array containing the request struct.
			params := [1]interface{}{args}
			if err = json.Unmarshal(*c.request.Params, &params); err != nil {
				c.err = &Error{
					Code:    E_INVALID_REQ,
					Message: err.Error(),
					Data:    c.request.Params,
				}
			}
		}
	}
	return c.err
}

// WriteResponse encodes the response and writes it to the ResponseWriter.
func (c *rawCodecRequest) WriteResponse(w io.Writer, reply interface{}) error {
	res := &serverResponse{
		Version: Version,
		Result:  reply,
		ID:      c.request.ID,
	}
	return c.writeServerResponse(w, res)
}

func (c *rawCodecRequest) WriteError(w io.Writer, status int, err error) error {
	jsonErr, ok := err.(*Error)
	if !ok {
		jsonErr = &Error{
			Code:    E_SERVER,
			Message: err.Error(),
		}
	}
	res := &serverResponse{
		Version: Version,
		Error:   jsonErr,
		ID:      c.request.ID,
	}
	return c.writeServerResponse(w, res)
}

func (c *rawCodecRequest) writeServerResponse(w io.Writer, res *serverResponse) error {
	// Id is null for notifications and they don't have a response.
	if c.request.ID != nil {
		encoder := json.NewEncoder(w)
		err := encoder.Encode(res)
		return err
	}
	return nil
}

// ----------------------------------------------------------------------------
// HTTPCodecRequest
// ----------------------------------------------------------------------------

// newCodecRequest returns a new CodecRequest.
func newHTTPCodecRequest(r *http.Request, encoder rpc.Encoder) rpc.CodecRequest {
	// Decode the request body and check if RPC method is valid.
	rawCodecRequest := newRawCodecRequest(r.Body)
	return &HTTPCodecRequest{raw: rawCodecRequest, encoder: encoder}
}

// HTTPCodecRequest decodes and encodes a single request.
type HTTPCodecRequest struct {
	raw     *rawCodecRequest
	encoder rpc.Encoder
}

// Method returns the RPC method for the current request.
//
// The method uses a dotted notation as in "Service.Method".
func (c *HTTPCodecRequest) Method() (string, error) {
	return c.raw.Method()
}

// ReadRequest fills the request object for the RPC method.
//
// ReadRequest parses request parameters in two supported forms in
// accordance with http://www.jsonrpc.org/specification#parameter_structures
//
// by-position: params MUST be an Array, containing the
// values in the Server expected order.
//
// by-name: params MUST be an Object, with member names
// that match the Server expected parameter names. The
// absence of expected names MAY result in an error being
// generated. The names MUST match exactly, including
// case, to the method's expected parameters.
func (c *HTTPCodecRequest) ReadRequest(args interface{}) error {
	return c.raw.ReadRequest(args)
}

// WriteResponse encodes the response and writes it to the ResponseWriter.
func (c *HTTPCodecRequest) WriteResponse(w http.ResponseWriter, reply interface{}) {
	rawCodecWriteErr := c.raw.WriteResponse(w, reply)
	c.writeServerResponse(w, rawCodecWriteErr)
}

// WriteError encodes the error response and writes it to the ResponseWriter
func (c *HTTPCodecRequest) WriteError(w http.ResponseWriter, status int, err error) {
	rawCodecWriteErr := c.raw.WriteError(w, status, err)
	c.writeServerResponse(w, rawCodecWriteErr)
}

func (c *HTTPCodecRequest) writeServerResponse(w http.ResponseWriter, rawCodecWriteErr error) {
	// Id is null for notifications and they don't have a response.
	if c.raw.request.ID != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		// Not sure in which case will this happen. But seems harmless.
		if rawCodecWriteErr != nil {
			rpc.WriteHTTPError(w, 400, rawCodecWriteErr.Error())
		}
	}
}

var _ rpc.CodecRequest = new(HTTPCodecRequest)
