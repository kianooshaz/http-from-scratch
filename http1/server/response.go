package server

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

type responseBodyWriter struct {
	proto       string
	conn        net.Conn
	sentHeaders bool
	headers     http.Header
}

func (r *responseBodyWriter) Header() http.Header {
	return r.headers
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if !r.sentHeaders {
		r.sendHeaders(http.StatusOK)
	}
	return r.conn.Write(b)
}

func (r *responseBodyWriter) WriteHeader(statusCode int) {
	if r.sentHeaders {
		slog.Warn(fmt.Sprintf("WriteHeader called twice, second time with: %d", statusCode))
		return
	}
	r.sendHeaders(statusCode)
}

func (r *responseBodyWriter) sendHeaders(statusCode int) {
	r.sentHeaders = true
	io.WriteString(r.conn, r.proto)
	r.conn.Write([]byte{' '})
	io.WriteString(r.conn, strconv.FormatInt(int64(statusCode), 10))
	r.conn.Write([]byte{' '})
	io.WriteString(r.conn, http.StatusText(statusCode))
	r.conn.Write([]byte{'\r', '\n'})
	for k, vals := range r.headers {
		for _, val := range vals {
			io.WriteString(r.conn, k)
			r.conn.Write([]byte{':', ' '})
			io.WriteString(r.conn, val)
			r.conn.Write([]byte{'\r', '\n'})
		}
	}
	r.conn.Write([]byte{'\r', '\n'})
}
