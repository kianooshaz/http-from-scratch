package server

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
)

var nlcf = []byte{0x0d, 0x0a}

type responseBodyWriter struct {
	req             *http.Request
	conn            net.Conn
	sentHeaders     bool
	headers         http.Header
	chunkedEncoding bool
	bodyBuffer      *bytes.Buffer
}

func (r *responseBodyWriter) Header() http.Header {
	return r.headers
}

func (r *responseBodyWriter) Write(b []byte) (int, error) {
	if !r.sentHeaders {
		if r.headers.Get("Content-Type") == "" {
			r.headers.Set("Content-Type", http.DetectContentType(b))
		}
		r.WriteHeader(http.StatusOK)
	}

	if r.chunkedEncoding {
		chunkSize := fmt.Sprintf("%x\r\n", len(b))
		if _, err := r.conn.Write([]byte(chunkSize)); err != nil {
			return 0, err
		}
	}

	n, err := r.conn.Write(b)
	if err != nil {
		return n, err
	}

	if r.chunkedEncoding {
		if _, err := r.conn.Write(nlcf); err != nil {
			return n, err
		}
	}

	return n, nil
}

func (r *responseBodyWriter) Flush() {
	if !r.sentHeaders {
		r.WriteHeader(http.StatusOK)
	}
	if flusher, ok := r.conn.(interface{ Flush() error }); ok {
		flusher.Flush()
	}
}

func (r *responseBodyWriter) flush() error {
	if r.chunkedEncoding {
		if _, err := r.conn.Write([]byte("0\r\n\r\n")); err != nil {
			return err
		}
	}

	r.writeBufferedBody()

	return nil
}

func (r *responseBodyWriter) WriteHeader(statusCode int) {
	if r.sentHeaders {
		slog.Warn(fmt.Sprintf("WriteHeader called twice, second time with: %d", statusCode))
		return
	}

	r.writeHeader(r.conn, r.req.Proto, r.headers, statusCode)
	r.sentHeaders = true
	r.writeBufferedBody()
}

func (r *responseBodyWriter) writeBufferedBody() {
	if r.bodyBuffer != nil {
		_, err := r.conn.Write(r.bodyBuffer.Bytes())
		if err != nil {
			slog.Error("Error writing buffered body", "err", err)
		}
		r.bodyBuffer = nil
	}
}

func (r *responseBodyWriter) writeHeader(conn io.Writer, proto string, headers http.Header, statusCode int) error {
	_, clSet := r.headers["Content-Length"]
	_, teSet := r.headers["Transfer-Encoding"]
	if !clSet && !teSet {
		r.chunkedEncoding = true
		r.headers.Set("Transfer-Encoding", "chunked")
	}

	if r.req.Close {
		r.headers.Set("Connection", "close")
	} else {
		r.headers.Set("Connection", "keep-alive")
	}

	if _, err := io.WriteString(conn, proto); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{' '}); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, strconv.FormatInt(int64(statusCode), 10)); err != nil {
		return err
	}
	if _, err := conn.Write([]byte{' '}); err != nil {
		return err
	}
	if _, err := io.WriteString(conn, http.StatusText(statusCode)); err != nil {
		return err
	}
	if _, err := conn.Write(nlcf); err != nil {
		return err
	}
	for k, vals := range headers {
		for _, val := range vals {
			if _, err := io.WriteString(conn, k); err != nil {
				return err
			}
			if _, err := conn.Write([]byte{':', ' '}); err != nil {
				return err
			}
			if _, err := io.WriteString(conn, val); err != nil {
				return err
			}
			if _, err := conn.Write(nlcf); err != nil {
				return err
			}
		}
	}
	if _, err := conn.Write(nlcf); err != nil {
		return err
	}
	return nil
}
