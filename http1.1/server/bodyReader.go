package server

import "io"

type noBody struct{}

func (noBody) Read([]byte) (int, error) { return 0, io.EOF }
func (noBody) Close() error             { return nil }

type bodyReader struct {
	reader io.Reader
}

func (r *bodyReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *bodyReader) Close() error {
	_, err := io.Copy(io.Discard, r.reader)
	return err
}
