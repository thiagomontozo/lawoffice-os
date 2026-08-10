package scanner

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func scanServer(t *testing.T, response string) (string, <-chan error) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	done := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		defer connection.Close()
		command := make([]byte, len("zINSTREAM\x00"))
		if _, readErr := io.ReadFull(connection, command); readErr != nil {
			done <- readErr
			return
		}
		if string(command) != "zINSTREAM\x00" {
			done <- errors.New("unexpected scanner command")
			return
		}
		for {
			var length uint32
			if readErr := binary.Read(connection, binary.BigEndian, &length); readErr != nil {
				done <- readErr
				return
			}
			if length == 0 {
				break
			}
			if _, readErr := io.CopyN(io.Discard, connection, int64(length)); readErr != nil {
				done <- readErr
				return
			}
		}
		_, writeErr := io.WriteString(connection, response+"\x00")
		done <- writeErr
	}()
	return listener.Addr().String(), done
}

func TestClamAVStreamingScan(t *testing.T) {
	for _, test := range []struct {
		name     string
		response string
		want     error
	}{
		{name: "clean", response: "stream: OK"},
		{name: "threat", response: "stream: Eicar-Signature FOUND", want: ErrThreat},
	} {
		t.Run(test.name, func(t *testing.T) {
			address, serverDone := scanServer(t, test.response)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := NewClamAV(address).Scan(ctx, strings.NewReader("sample legal content"))
			if !errors.Is(err, test.want) {
				t.Fatalf("scan error %v, want %v", err, test.want)
			}
			if serverErr := <-serverDone; serverErr != nil {
				t.Fatalf("fake ClamAV server: %v", serverErr)
			}
		})
	}
}

func TestClamAVHealth(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			done <- acceptErr
			return
		}
		defer connection.Close()
		command := make([]byte, len("zPING\x00"))
		_, readErr := io.ReadFull(connection, command)
		if readErr == nil && string(command) != "zPING\x00" {
			readErr = errors.New("unexpected health command")
		}
		if readErr == nil {
			_, readErr = io.WriteString(connection, "PONG\x00")
		}
		done <- readErr
	}()
	if err = NewClamAV(listener.Addr().String()).Health(context.Background()); err != nil {
		t.Fatalf("health: %v", err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}
