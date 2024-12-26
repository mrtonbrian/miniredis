package miniredis

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type RESPDataType int16

const (
	SimpleString RESPDataType = iota
	BulkString
	Integer
	Array
)

type RESPReader struct {
	reader bufio.Reader
}

type RESPWriter struct {
	writer bufio.Writer
}

func NewRESPReader(conn io.Reader) *RESPReader {
	return &RESPReader{
		reader: *bufio.NewReader(conn),
	}
}

func NewRESPWriter(conn io.Writer) *RESPWriter {
	return &RESPWriter{
		writer: *bufio.NewWriter(conn),
	}
}

type RESPData interface{ DataType() RESPDataType }

type RESPSimpleString struct{ data string }

func (s *RESPSimpleString) DataType() RESPDataType { return SimpleString }

type RESPBulkString struct{ data []byte }

func (s *RESPBulkString) DataType() RESPDataType { return BulkString }

type RESPInteger struct{ data int64 }

func (s *RESPInteger) DataType() RESPDataType { return Integer }

type RESPArray struct{ data []RESPData }

func (s *RESPArray) DataType() RESPDataType { return Array }

type RESPCommandType int16

const (
	SET RESPCommandType = iota
	GET
	ECHO
)

type RESPCommand struct {
	Type RESPCommandType
	Args []RESPData
}

func (r *RESPReader) ReadInteger() (*RESPInteger, error) {
	line, _, err := r.reader.ReadLine()
	if err != nil {
		return &RESPInteger{}, fmt.Errorf("reading integer: %w", err)
	}

	val, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return &RESPInteger{}, fmt.Errorf("parsing integer: %w", err)
	}

	return &RESPInteger{data: val}, nil
}

func (r *RESPReader) ReadBulkString() (*RESPBulkString, error) {
	length, err := r.ReadInteger()
	if err != nil {
		return &RESPBulkString{}, fmt.Errorf("reading length of bulk string: %w", err)
	}

	if length.data < 0 {
		return &RESPBulkString{}, nil
	}

	bulk := make([]byte, length.data)
	_, err = r.reader.Read(bulk)
	if err != nil {
		return &RESPBulkString{}, fmt.Errorf("reading bulk string: %w", err)
	}

	// Read \r\n at the end of the bulk string

	r.reader.ReadByte()
	r.reader.ReadByte()

	// log.Printf("Read %s\n", string(bulk))
	return &RESPBulkString{data: bulk}, nil
}

func (r *RESPReader) ReadSimpleString() (*RESPSimpleString, error) {
	out, _, err := r.reader.ReadLine()
	if err != nil {
		return &RESPSimpleString{}, fmt.Errorf("reading simple string: %w", err)
	}
	// log.Printf("Read %s\n", string(out))
	return &RESPSimpleString{data: string(out)}, nil
}

func (r *RESPReader) ReadArray() (*RESPArray, error) {
	length, err := r.ReadInteger()
	if err != nil {
		return &RESPArray{}, fmt.Errorf("reading length of array: %w", err)
	}

	if length.data < 0 {
		return &RESPArray{}, nil
	}

	arr := make([]RESPData, length.data)

	for i := 0; i < int(length.data); i++ {
		data, err := r.ReadValue()
		if err != nil {
			return &RESPArray{}, fmt.Errorf("reading array element: %w", err)
		}
		arr[i] = data
	}

	return &RESPArray{data: arr}, nil
}

func (r *RESPReader) ReadValue() (RESPData, error) {
	_type, err := r.reader.ReadByte()
	if err != nil {
		return nil, err
	}
	// log.Printf("Recieved %s", string([]byte{_type}))
	switch _type {
	case '+':
		return r.ReadSimpleString()
	case ':':
		return r.ReadInteger()
	case '$':
		return r.ReadBulkString()
	case '*':
		return r.ReadArray()
	default:
		return nil, fmt.Errorf("unknown data type %b", _type)
	}
}

func ParseCommand(commandArray *RESPArray) (RESPCommand, error) {
	var commandType RESPCommandType
	var commandName string
	firstArg := commandArray.data[0]
	switch v := firstArg.(type) {
	case *RESPSimpleString:
		commandName = v.data
	case *RESPBulkString:
		commandName = string(v.data)
	default:
		return RESPCommand{}, fmt.Errorf("unknown command type %T", firstArg)
	}

	commandName = strings.ToUpper(commandName)

	switch commandName {
	case "SET":
		commandType = SET
	case "GET":
		commandType = GET
	case "ECHO":
		commandType = ECHO
	default:
		return RESPCommand{}, fmt.Errorf("unknown command %s", commandName)
	}

	return RESPCommand{Type: commandType, Args: commandArray.data[1:]}, nil
}

func ExtractString(data *RESPData) (string, error) {
	switch v := (*data).(type) {
	case *RESPSimpleString:
		return v.data, nil
	case *RESPBulkString:
		return string(v.data), nil
	default:
		return "", fmt.Errorf("expected string, got %T", data)
	}
}

func ExtractByteSlice(data *RESPData) ([]byte, error) {
	switch v := (*data).(type) {
	case *RESPSimpleString:
		return []byte(v.data), nil
	case *RESPBulkString:
		return v.data, nil
	default:
		return nil, fmt.Errorf("expected string, got %T", data)
	}
}

func (w *RESPWriter) WriteBulkString(b []byte) error {
	if b == nil {
		_, err := w.writer.Write([]byte("$-1\r\n"))
		return err
	}
	// log.Printf("Writing %s\n", string(b))
	w.writer.WriteString("$")
	w.writer.WriteString(strconv.Itoa(len(b)))
	w.writer.WriteString("\r\n")
	w.writer.Write(b)
	w.writer.WriteString("\r\n")
	return nil
}

func (w *RESPWriter) WriteInteger(i int64) error {
	// log.Printf("Writing %d\n", i)
	w.writer.WriteString(":")
	w.writer.WriteString(strconv.FormatInt(i, 10))
	w.writer.WriteString("\r\n")
	return nil
}

func (w *RESPWriter) WriteError(err error) error {
	_, err = fmt.Fprintf(&w.writer, "-ERR %s\r\n", err.Error())
	return err
}

// WriteValue determines the type of MiniRedisData and writes appropriate RESP format
func (w *RESPWriter) WriteValue(v MiniRedisData) error {
	switch data := v.(type) {
	case *StringData:
		return w.WriteBulkString(data.data)
	case *IntegerData:
		return w.WriteInteger(data.data)
	default:
		return fmt.Errorf("unknown data type: %T", v)
	}
}
