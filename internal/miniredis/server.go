package miniredis

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

var store = NewConcurrentMap[string, MiniRedisObject]()

func handleSet(args []RESPData) (MiniRedisData, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("SET command requires at least 2 arguments")
	}

	key, err := ExtractString(&args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	value, err := ExtractByteSlice(&args[1])
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	byteSliceCopy := make([]byte, len(value))
	copy(byteSliceCopy, []byte(value))
	stringData := &StringData{data: byteSliceCopy}
	obj := MiniRedisObject{
		data:   stringData,
		expiry: time.Time{},
	}

	store.Set(&key, &obj)
	return stringData, nil
}

func handleGet(args []RESPData) (MiniRedisData, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("GET command requires exactly 1 argument")
	}

	key, err := ExtractString(&args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	value, exists := store.Get(&key)

	if !exists {
		return &StringData{data: nil}, nil
	}

	if !value.expiry.IsZero() && time.Now().After(value.expiry) {
		store.Delete(&key)
		return &StringData{data: nil}, nil
	}

	return value.data, nil
}

func handleEcho(args []RESPData) (MiniRedisData, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("ECHO command requires exactly 1 argument")
	}

	arg, err := ExtractByteSlice(&args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid echo param: %w", err)
	}

	return &StringData{data: arg}, nil
}
func HandleConnection(conn net.Conn) error {
	defer conn.Close()

	// Initialize RESPReader with 4kb buffer
	respReader := NewRESPReader(conn, RESP_READER_INITIAL_BUF_SIZE)
	respWriter := NewRESPWriter(conn, RESP_WRITER_INITIAL_BUF_SIZE)

	for {
		commands, err := respReader.ReadCommands()
		if err != nil {
			// For a net.Conn, io.EOF is only returned if there's no data that was read
			// so it's safe to just exit
			if err == io.EOF {
				return nil
			}

			return fmt.Errorf("reading commands: %w", err)
		}

		// Process all commands we read
		for _, cmd := range commands {
			result, handlerErr := dispatchCommand(&cmd)

			if handlerErr != nil {
				if err := respWriter.WriteError(handlerErr); err != nil {
					return fmt.Errorf("writing error: %w", err)
				}
				continue
			}

			if err := respWriter.WriteValue(result); err != nil {
				return fmt.Errorf("writing response: %w", err)
			}
		}

		// Flush all responses together
		if err := respWriter.writer.Flush(); err != nil {
			return fmt.Errorf("flushing: %w", err)
		}
	}
}

func dispatchCommand(cmd *RESPCommand) (MiniRedisData, error) {
	switch cmd.Type {
	case SET:
		return handleSet(cmd.Args)
	case GET:
		return handleGet(cmd.Args)
	case ECHO:
		return handleEcho(cmd.Args)
	default:
		return nil, fmt.Errorf("unsupported command: %v", cmd.Type)
	}
}

func StartServer(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to bind to port 9092: %v", err)
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Printf("failed to accept connection: %v", err)
			continue
		}

		go func() {
			if err := HandleConnection(conn); err != nil {
				log.Printf("error handling connection: %v", err)
			}
		}()
	}
}
