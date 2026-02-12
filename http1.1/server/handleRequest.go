package server

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (s *Server) handleRequest(conn net.Conn) (bool, error) {
	// Limit headers to 1MB
	limitReader := io.LimitReader(conn, 1*1024*1024).(*io.LimitedReader)
	reader := bufio.NewReader(limitReader)

	reqLineBytes, _, err := reader.ReadLine()
	if err != nil {
		return true, fmt.Errorf("read request line error: %w", err)
	}
	reqLine := string(reqLineBytes)

	req := new(http.Request)
	var found bool

	req.Method, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return true, errors.New("invalid method")
	}
	if !methodValid(req.Method) {
		return true, errors.New("invalid method")
	}

	req.RequestURI, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return true, errors.New("invalid path")
	}
	if req.URL, err = url.ParseRequestURI(req.RequestURI); err != nil {
		return true, fmt.Errorf("invalid path: %w", err)
	}

	req.Proto = reqLine
	req.ProtoMajor, req.ProtoMinor, found = parseProtocol(req.Proto)
	if !found {
		return true, errors.New("invalid protocol")
	}

	req.Header = make(http.Header)
	for {
		line, _, err := reader.ReadLine()
		if err != nil && err != io.EOF {
			return true, err
		} else if err != nil {
			break
		}
		if len(line) == 0 {
			break
		}

		k, v, ok := bytes.Cut(line, []byte{':'})
		if !ok {
			return true, errors.New("invalid header")
		}
		req.Header.Add(strings.ToLower(string(k)), strings.TrimLeft(string(v), " "))
	}

	if _, ok := req.Header["Host"]; !ok {
		return true, errors.New("required 'Host' header not found")
	}

	switch strings.ToLower(req.Header.Get("Connection")) {
	case "keep-alive", "":
		req.Close = false
	case "close":
		req.Close = true
	}

	limitReader.N = math.MaxInt64

	ctx := context.Background()
	ctx = context.WithValue(ctx, http.LocalAddrContextKey, conn.LocalAddr())
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()
	contentLength, err := parseContentLength(req.Header.Get("Content-Length"))
	if err != nil {
		return true, err
	}
	req.ContentLength = contentLength
	isChunked := req.Header.Get("Transfer-Encoding") == "chunked"
	if req.ContentLength == 0 && !isChunked {
		req.Body = noBody{}
	} else {
		if isChunked {
			req.Body = &chunkedBodyReader{
				reader: reader,
			}
		} else {
			req.Body = &bodyReader{
				reader: io.LimitReader(reader, req.ContentLength),
			}
		}
	}

	req.RemoteAddr = conn.RemoteAddr().String()

	w := &responseBodyWriter{
		req:     req,
		conn:    conn,
		headers: make(http.Header),
	}

	s.Handler.ServeHTTP(w, req.WithContext(ctx))
	if err := w.flush(); err != nil {
		return true, nil
	}
	return req.Close, nil
}

func parseContentLength(headerval string) (int64, error) {
	if headerval == "" {
		return 0, nil
	}
	return strconv.ParseInt(headerval, 10, 64)
}

func parseProtocol(proto string) (int, int, bool) {
	switch proto {
	case "HTTP/1.0":
		return 1, 0, true
	case "HTTP/1.1":
		return 1, 1, true
	}
	return 0, 0, false
}

func methodValid(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return true
	}
	return false
}
