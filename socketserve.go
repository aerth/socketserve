/*
* The MIT License (MIT)
*
* Copyright (c) 2017  aerth <aerth@riseup.net>
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in all
* copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
* SOFTWARE.
 */

// Package socketserve serves http on a unix socket
package main

import (
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

var (
	socket = flag.String("s", os.Getenv("SOCKET")+"web.sock", "location of network socket to listen on")
)

func usage() {
	println("usage:")
	println("\tsockserve [-flag] <directory>")
	println("flags:")
	flag.PrintDefaults()
}

func init() {
	flag.Usage = usage
}

func main() {
	println("socketserve v98")
	flag.Parse()
	if len(flag.Args()) != 1 {
		usage()
		os.Exit(111)
	}
	dir := flag.Args()[0]
	server := New(dir)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc)
	go func() {
		sig := <-sigc
		if server != nil {
			println("got signal:", sig.Signal)
			server.close()
		}
		os.Exit(111)
	}()

	go func() {
		<-time.After(time.Second)
		wd, _ := os.Getwd()
		println("listening on unix socket:", filepath.Join(wd, *socket))
	}()
	server.Serve(*socket)

}

type Server struct {
	handler    http.Handler
	listener   net.Listener
	socketpath string
}

func New(dir string) *Server {
	s := new(Server)

	// check if dir or single file or nothing
	stat, err := os.Stat(dir)
	if err != nil {
		println("fatal:", err.Error())
		os.Exit(111)
	}
	if stat.IsDir() {
		s.handler = http.FileServer(http.Dir(dir))
	} else {
		println("fatal:", "need directory as single argument")
	}
	return s
}

func (s Server) Serve(address string) {
	defer os.Remove(address)
ServeUnix:
	// Look up address
	socketAddress, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		println(err.Error())
		return
	}
	ulistener, err := net.ListenUnix("unix", socketAddress)
	if err != nil {
		if strings.Contains(err.Error(), "already in use") {
			os.Remove(address)
			goto ServeUnix
		}
		println(err.Error())
		return
	}
	s.listener = ulistener
	err = http.Serve(s.listener, s.handler)
	if err != nil {
		if !strings.Contains(err.Error(), "use of closed") {
			println(err.Error)
		}
	}
}

func (s *Server) close() {
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketpath)
}
