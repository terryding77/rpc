// Copyright 2009 The Go Authors. All rights reserved.
// Copyright 2013 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/terryding77/rpc"
	"github.com/terryding77/rpc/jsonrpc2"
)

type Counter struct {
	Count int
}

type IncrReq struct {
	Delta int
}

type EmptyResponse struct {
}

// Notification.
func (c *Counter) Incr(r *http.Request, req *IncrReq, res *EmptyResponse) error {
	log.Printf("<- Incr %+v", *req)
	c.Count += req.Delta
	return nil
}

type GetReq struct {
}

func (c *Counter) Get(r *http.Request, req *GetReq, res *Counter) error {
	log.Printf("<- Get %+v", *req)
	*res = *c
	log.Printf("-> %v", *res)
	return nil
}

func main() {
	address := flag.String("address", ":65534", "")
	s := rpc.NewServer()
	s.RegisterCodec(jsonrpc2.NewCustomCodec(&rpc.CompressionSelector{}), "application/json")
	s.RegisterService(new(Counter), "")
	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./"))))
	http.Handle("/jsonrpc/", s)
	log.Fatal(http.ListenAndServe(*address, nil))
}
