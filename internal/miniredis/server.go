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
	// log.Println("Encountered Set")
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

	stringData := &StringData{data: []byte(value)}
	obj := MiniRedisObject{
		data:   stringData,
		expiry: time.Time{},
	}

	store.Set(&key, &obj)
	return stringData, nil
}

func handleGet(args []RESPData) (MiniRedisData, error) {
	// log.Println("Encountered Get")
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
	// log.Println("Encountered Echo")
	if len(args) != 1 {
		return nil, fmt.Errorf("ECHO command requires exactly 1 argument")
	}

	arg, err := ExtractByteSlice(&args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid echo param: %w", err)
	}

	return &StringData{data: arg}, nil
}

func handleConnection(conn net.Conn) error {
	defer conn.Close()
	respReader := NewRESPReader(conn)
	respWriter := NewRESPWriter(conn)

	for {
		// log.Println("Handling new command")
		value, err := respReader.ReadValue()
		if err != nil {
			if err == io.EOF {
				// Client closed connection normally
				return nil
			}
			return fmt.Errorf("reading value from connection: %w", err)
		}

		// Check that command is an array
		commandArray, ok := value.(*RESPArray)
		if !ok {
			return fmt.Errorf("expected array, got %T", value)
		}

		// log.Println("Read command")

		// Parse the command
		command, err := ParseCommand(commandArray)
		if err != nil {
			return fmt.Errorf("parsing command: %w", err)
		}

		// log.Println("Parsed command")

		var result MiniRedisData
		var handlerErr error

		switch command.Type {
		case SET:
			result, handlerErr = handleSet(command.Args)
		case GET:
			result, handlerErr = handleGet(command.Args)
		case ECHO:
			result, handlerErr = handleEcho(command.Args)
		default:
			handlerErr = fmt.Errorf("unsupported command")
		}

		// log.Println("Finished handling command")

		if handlerErr != nil {
			if err := respWriter.WriteError(handlerErr); err != nil {
				return fmt.Errorf("writing error response: %w", err)
			}
			continue
		}

		if err := respWriter.WriteValue(result); err != nil {
			return fmt.Errorf("writing response: %w", err)
		}

		respWriter.writer.Flush()
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
			// log.Printf("failed to accept connection: %v", err)
			continue
		}

		go func() {
			if err := handleConnection(conn); err != nil {
				// log.Printf("error handling connection: %v", err)
			}
		}()
	}
}
