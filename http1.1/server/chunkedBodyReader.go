package server

import (
	"bufio"
	"errors"
	"io"
	"strconv"
	"strings"
)

type chunkedBodyReader struct {
	reader         *bufio.Reader
	bytesRemaining int64 // bytes left in current chunk
	stickyErr      error // persistent read error
}

func (r *chunkedBodyReader) Read(p []byte) (int, error) {
	if r.stickyErr != nil {
		return 0, r.stickyErr
	}

	// Need a new chunk?
	if r.bytesRemaining == 0 {
		size, err := r.nextChunkSize()
		if err != nil {
			r.stickyErr = err
			return 0, err
		}
		r.bytesRemaining = size
	}

	// Last chunk (size 0) => EOF
	if r.bytesRemaining == 0 {
		return 0, io.EOF
	}

	// Limit read to remaining chunk size
	if int64(len(p)) > r.bytesRemaining {
		p = p[:r.bytesRemaining]
	}

	n, err := r.reader.Read(p)
	r.bytesRemaining -= int64(n)

	// If chunk ended, consume trailing CRLF
	if r.bytesRemaining == 0 && err == nil {
		if err := r.consumeCRLF(); err != nil {
			r.stickyErr = err
			return n, err
		}
	}

	r.stickyErr = err
	return n, err
}

func (r *chunkedBodyReader) nextChunkSize() (int64, error) {
	line, err := r.readCRLFLine()
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseInt(strings.TrimSpace(line), 16, 64)
	if err != nil {
		return 0, err
	}

	// Final chunk → read trailers
	if size == 0 {
		for {
			line, err := r.readCRLFLine()
			if err != nil {
				return 0, err
			}
			if line == "" {
				break
			}
		}
	}

	return size, nil
}

func (r *chunkedBodyReader) readCRLFLine() (string, error) {
	var line []byte

	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' {
			break
		}
		line = append(line, b)
	}

	return strings.TrimRight(string(line), "\r"), nil
}

func (r *chunkedBodyReader) consumeCRLF() error {
	if b, err := r.reader.ReadByte(); err != nil || b != '\r' {
		if err != nil {
			return err
		}
		return errors.New("missing CR after chunk")
	}
	if b, err := r.reader.ReadByte(); err != nil || b != '\n' {
		if err != nil {
			return err
		}
		return errors.New("missing LF after chunk")
	}
	return nil
}

func (r *chunkedBodyReader) Close() error {
	_, err := io.Copy(io.Discard, r)
	return err
}
