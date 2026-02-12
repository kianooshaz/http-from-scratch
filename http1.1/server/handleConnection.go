package server

import (
	"errors"
	"io"
	"net"
)

func (s *Server) handleConnection(conn net.Conn) error {
	defer conn.Close()
	for {
		// handleRequest does the work of reading and responding
		shouldClose, err := s.handleRequest(conn)
		if err != nil {
			// io.EOF is a normal way for a persistent connection to end.
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if shouldClose {
			return nil // Client requested a close, so we exit the loop.
		}
	}
}
