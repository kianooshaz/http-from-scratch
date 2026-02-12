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
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

func (s *Server) handleConnection(conn net.Conn) error {
	defer conn.Close()

	// Limit headers to 1MB
	limitReader := io.LimitReader(conn, 1*1024*1024).(*io.LimitedReader)
	reader := bufio.NewReader(limitReader)
	headerReader := textproto.NewReader(reader)

	// Read the request line: GET /path/to/index.html HTTP/1.0
	reqLine, err := headerReader.ReadLine()
	if err != nil {
		return fmt.Errorf("read request line error: %w", err)
	}

	req := new(http.Request)
	var found bool

	// Parse Method: GET/POST/PUT/DELETE/etc
	req.Method, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return errors.New("invalid method")
	}
	if !methodValid(req.Method) {
		return errors.New("invalid method")
	}

	// Parse Request URI
	req.RequestURI, reqLine, found = strings.Cut(reqLine, " ")
	if !found {
		return errors.New("invalid path")
	}
	if req.URL, err = url.ParseRequestURI(req.RequestURI); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Parse protocol version "HTTP/1.0"
	req.Proto = reqLine
	req.ProtoMajor, req.ProtoMinor, found = parseProtocol(req.Proto)
	if !found {
		return errors.New("invalid proto")
	}

	// Parse headers
	req.Header = make(http.Header)
	for {
		line, err := headerReader.ReadLineBytes()
		if err != nil && err != io.EOF {
			return err
		} else if err != nil {
			break
		}
		if len(line) == 0 {
			break
		}

		k, v, ok := bytes.Cut(line, []byte{':'})
		if !ok {
			return errors.New("invalid header")
		}
		req.Header.Add(strings.ToLower(string(k)), strings.TrimLeft(string(v), " "))
	}

	// Unbound the limit after we've read the headers since the body can be any size
	limitReader.N = math.MaxInt64

	ctx := context.Background()
	ctx = context.WithValue(ctx, http.LocalAddrContextKey, conn.LocalAddr())
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()
	contentLength, err := parseContentLength(req.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	req.ContentLength = contentLength
	if req.ContentLength == 0 {
		req.Body = noBody{}
	} else {
		req.Body = &bodyReader{reader: io.LimitReader(reader, req.ContentLength)}
	}

	req.RemoteAddr = conn.RemoteAddr().String()
	req.Close = true // this is always true for HTTP/1.0

	w := &responseBodyWriter{
		// We hard-code this because this is a HTTP/1.0 server.
		// Web servers will make requests with HTTP/1.1 but
		// we're saying that we only support HTTP/1.0.
		proto:   "HTTP/1.0",
		conn:    conn,
		headers: make(http.Header),
	}

	// Finally, call our http.Handler!
	s.Handler.ServeHTTP(w, req.WithContext(ctx))
	if !w.sentHeaders {
		w.sendHeaders(http.StatusOK)
	}
	return nil
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
