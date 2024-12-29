package miniredis

import (
	"bufio"
	"errors"
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
	reader     io.Reader
	buffer     []byte
	readIndex  int
	writeIndex int
}

type RESPWriter struct {
	writer bufio.Writer
}

func NewRESPReader(conn io.Reader, initialBufferSize int) *RESPReader {
	return &RESPReader{
		reader: conn,
		buffer: make([]byte, initialBufferSize),
	}
}

func NewRESPWriter(conn io.Writer) *RESPWriter {
	return &RESPWriter{
		writer: *bufio.NewWriter(conn),
	}
}

type RESPData interface {
	DataType() RESPDataType
	Dump() string // Mainly for debugging
}

type RESPSimpleString struct{ data string }

func (s *RESPSimpleString) DataType() RESPDataType { return SimpleString }
func (s *RESPSimpleString) Dump() string           { return s.data }

type RESPBulkString struct{ data []byte }

func (s *RESPBulkString) DataType() RESPDataType { return BulkString }
func (s *RESPBulkString) Dump() string           { return string(s.data) }

type RESPInteger struct{ data int64 }

func (s *RESPInteger) DataType() RESPDataType { return Integer }
func (s *RESPInteger) Dump() string           { return strconv.FormatInt(s.data, 10) }

type RESPArray struct{ data []RESPData }

func (s *RESPArray) DataType() RESPDataType { return Array }
func (s *RESPArray) Dump() string {
	var stringsArr []string
	for _, elem := range s.data {
		stringsArr = append(stringsArr, elem.Dump())
	}
	return fmt.Sprintf("[%s]", strings.Join(stringsArr, ","))
}

func DumpRESPDataArray(lst []RESPData) string {
	var stringsArr []string
	for _, elem := range lst {
		stringsArr = append(stringsArr, elem.Dump())
	}
	return fmt.Sprintf("[%s]", strings.Join(stringsArr, ","))
}

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

var ErrIncompleteRESPValue = errors.New("incomplete RESP value")

// Reads commands in from r.reader, handles buffering. Meant to be called in a loop
func (r *RESPReader) ReadCommands() ([]RESPCommand, error) {
	r.shiftBuffer()

	r.growBuffer()

	numReadBytes, err := r.reader.Read(r.buffer[r.writeIndex:])

	if err != nil {
		// We've read to the end of the stream, just parse commands in buffer
		if err == io.EOF {
			commands, parseErr := r.parseBufferCommands()
			if parseErr != nil {
				return commands, parseErr
			}

			// Return EOF to caller
			return commands, io.EOF
		}

		return nil, fmt.Errorf("reading from reader: %w", err)
	}

	r.writeIndex += numReadBytes

	return r.parseBufferCommands()
}

// Shifts buffer to start at 0
func (r *RESPReader) shiftBuffer() {
	// Read everything in buffer, just reset indices
	if r.readIndex == r.writeIndex {
		r.readIndex = 0
		r.writeIndex = 0
		return
	}

	// If we've consumed something, then shift everything over
	if r.readIndex > 0 {
		leftoverLength := r.writeIndex - r.readIndex
		copy(r.buffer, r.buffer[r.readIndex:r.writeIndex])
		r.readIndex = 0
		r.writeIndex = leftoverLength
	}
}

// Grows buffer (double size) if it's too full (< 25% space remaining).
// Written to handle case where r.readIndex is not 0 (although only called in the case where it definitely is).
func (r *RESPReader) growBuffer() {
	size := len(r.buffer)
	used := r.writeIndex - r.readIndex
	free := size - used

	if free <= size/4 {
		newBuf := make([]byte, size*2)
		copy(newBuf, r.buffer[r.readIndex:r.writeIndex])
		r.buffer = newBuf
		r.readIndex = 0
		r.writeIndex = used
	}
}

// Parse all *complete* buffer commands, return as a slice
func (r *RESPReader) parseBufferCommands() ([]RESPCommand, error) {
	var commands []RESPCommand

	for {
		// Read everything in the buffer, so just exit loop
		if r.readIndex >= r.writeIndex {
			break
		}

		subBuffer := r.buffer[r.readIndex:r.writeIndex]
		value, consumed, err := parseSingleValue(subBuffer)
		if err != nil {
			// Ran out of commands in the buffer, just exit loop
			if err == ErrIncompleteRESPValue {
				break
			}

			return commands, fmt.Errorf("parsing buffer command: %w", err)
		}

		r.readIndex += consumed

		val, ok := value.(*RESPArray)
		if !ok {
			return commands, fmt.Errorf("error casting value to RESPArray - value datatype is %#v", value.DataType())
		}

		cmd, err := ParseCommand(val)
		if err != nil {
			return commands, fmt.Errorf("error parsing command: %w", err)
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// Parses single RESP value from buffer
func parseSingleValue(buf []byte) (RESPData, int, error) {
	if len(buf) == 0 {
		return nil, 0, ErrIncompleteRESPValue
	}

	// buffer[0] is the type
	switch buf[0] {
	case ':':
		return parseInteger(buf)
	case '$':
		return parseBulkString(buf)
	case '+':
		return parseSimpleString(buf)
	case '*':
		return parseArray(buf)
	default:
		return nil, 0, fmt.Errorf("unknown RESP type byte: %q", buf[0])
	}
}

// Parses integer into RESPInteger.
// Expects X1234\r\nXXXXX (note that it ignores the first character)
// Returns RESPInteger, how many bytes were consumed, and any errors
// Returns IncompleteRESPValueError if not enough data inside buffer to parse integer
func parseInteger(buf []byte) (*RESPInteger, int, error) {
	// Find where integer ends
	end := CLRFIndex(buf)

	if end == -1 {
		// There's more data outside the buffer that needs to be read in,
		// Return IncompleteRESPValueError
		return &RESPInteger{data: 0}, 0, ErrIncompleteRESPValue
	}

	intPart := string(buf[1:end])
	data, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return &RESPInteger{data: 0}, 0, err
	}

	// end + 2 bytes consumed in general
	return &RESPInteger{data: data}, end + 2, nil
}

// Parses integer into RESPBulkString. Expects $len\r\nBULK\r\n
// Returns RESPBulkString, how many bytes were consumed, and any errors.
// Returns IncompleteRESPValueError if not enough data inside buffer to parse integer
func parseBulkString(buf []byte) (*RESPBulkString, int, error) {
	totalConsumed := 0

	length, consumed, err := parseInteger(buf)
	if err != nil {
		return &RESPBulkString{data: nil}, 0, err
	}

	totalConsumed += consumed
	parsedLength := length.data

	if parsedLength < 0 {
		return &RESPBulkString{data: nil}, totalConsumed, nil
	}

	// Entire bulkstring does not fit in buffer
	if int64(len(buf)) < parsedLength+int64(consumed)+2 {
		return &RESPBulkString{data: nil}, 0, ErrIncompleteRESPValue
	}

	startInd := totalConsumed
	data := buf[startInd : startInd+int(parsedLength)]
	totalConsumed += int(parsedLength)

	// Otherwise, return bulk string
	return &RESPBulkString{data: data}, totalConsumed + 2, nil
}

// Parses simple string into RESPSimpleString
// Expects +ASDF\r\n
func parseSimpleString(buf []byte) (*RESPSimpleString, int, error) {
	end := CLRFIndex(buf)

	if end == -1 {
		return &RESPSimpleString{data: ""}, 0, ErrIncompleteRESPValue
	}

	return &RESPSimpleString{data: string(buf[1:end])}, end + 2, nil
}

// Parses array into RESPArray
// Expects *len\r\nELEM1\r\nELEM2\r\n...\r\nELEMN\r\n
func parseArray(buf []byte) (*RESPArray, int, error) {
	totalConsumed := 0

	length, consumed, err := parseInteger(buf)
	if err != nil {
		return &RESPArray{data: nil}, 0, err
	}

	totalConsumed += consumed
	parsedLength := length.data

	if parsedLength < 0 {
		return &RESPArray{data: nil}, totalConsumed, nil
	}

	arr := make([]RESPData, parsedLength)

	for i := 0; i < int(parsedLength); i++ {
		elem, consumed, err := parseSingleValue(buf[totalConsumed:])

		if err != nil {
			return &RESPArray{data: nil}, 0, err
		}

		totalConsumed += consumed
		arr[i] = elem
	}

	return &RESPArray{data: arr}, totalConsumed, nil
}

// Returns first index where \r\n is inside the string
func CLRFIndex(buf []byte) int {
	for i := 0; i < len(buf)-1; i++ {
		if buf[i] == '\r' && buf[i+1] == '\n' {
			return i
		}
	}

	return -1
}

func ParseCommand(commandArray *RESPArray) (RESPCommand, error) {
	var commandType RESPCommandType
	var commandName string

	if len(commandArray.data) == 0 {
		return RESPCommand{}, fmt.Errorf("command array length 0")
	}

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
	w.writer.WriteString("$")
	w.writer.WriteString(strconv.Itoa(len(b)))
	w.writer.WriteString("\r\n")
	w.writer.Write(b)
	w.writer.WriteString("\r\n")
	return nil
}

func (w *RESPWriter) WriteInteger(i int64) error {
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
