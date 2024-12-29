package miniredis

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestParseInteger(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectedVal   int64
		expectedBytes int
		expectError   bool
	}{
		{
			name:          "basic integer",
			input:         []byte(":1234\r\n"),
			expectedVal:   1234,
			expectedBytes: 7,
			expectError:   false,
		},
		{
			name:          "negative integer",
			input:         []byte(":-123\r\n"),
			expectedVal:   -123,
			expectedBytes: 7,
			expectError:   false,
		},
		{
			name:          "zero",
			input:         []byte(":0\r\n"),
			expectedVal:   0,
			expectedBytes: 4,
			expectError:   false,
		},
		{
			name:          "incomplete data",
			input:         []byte(":123"),
			expectedVal:   0,
			expectedBytes: 0,
			expectError:   true,
		},
		{
			name:          "invalid integer",
			input:         []byte(":abc\r\n"),
			expectedVal:   0,
			expectedBytes: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, bytesConsumed, err := parseInteger(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.data != tt.expectedVal {
				t.Errorf("got value %d, want %d", result.data, tt.expectedVal)
			}

			if bytesConsumed != tt.expectedBytes {
				t.Errorf("got %d bytes consumed, want %d", bytesConsumed, tt.expectedBytes)
			}
		})
	}
}

func TestParseSimpleString(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectedVal   string
		expectedBytes int
		expectError   bool
	}{
		{
			name:          "basic string",
			input:         []byte("+OK\r\n"),
			expectedVal:   "OK",
			expectedBytes: 5,
			expectError:   false,
		},
		{
			name:          "empty string",
			input:         []byte("+\r\n"),
			expectedVal:   "",
			expectedBytes: 3,
			expectError:   false,
		},
		{
			name:          "incomplete string",
			input:         []byte("+OK"),
			expectedVal:   "",
			expectedBytes: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, bytesConsumed, err := parseSimpleString(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.data != tt.expectedVal {
				t.Errorf("got value %q, want %q", result.data, tt.expectedVal)
			}

			if bytesConsumed != tt.expectedBytes {
				t.Errorf("got %d bytes consumed, want %d", bytesConsumed, tt.expectedBytes)
			}
		})
	}
}

func TestParseBulkString(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectedVal   []byte
		expectedBytes int
		expectError   bool
	}{
		{
			name:          "basic bulk string",
			input:         []byte("$5\r\nhello\r\n"),
			expectedVal:   []byte("hello"),
			expectedBytes: 11,
			expectError:   false,
		},
		{
			name:          "empty bulk string",
			input:         []byte("$0\r\n\r\n"),
			expectedVal:   []byte{},
			expectedBytes: 6,
			expectError:   false,
		},
		{
			name:          "null bulk string",
			input:         []byte("$-1\r\n"),
			expectedVal:   nil,
			expectedBytes: 5,
			expectError:   false,
		},
		{
			name:          "incomplete bulk string",
			input:         []byte("$5\r\nhel"),
			expectedVal:   nil,
			expectedBytes: 0,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, bytesConsumed, err := parseBulkString(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if string(result.data) != string(tt.expectedVal) {
				t.Errorf("got value %q, want %q", string(result.data), string(tt.expectedVal))
			}

			if bytesConsumed != tt.expectedBytes {
				t.Errorf("got %d bytes consumed, want %d", bytesConsumed, tt.expectedBytes)
			}
		})
	}
}

func TestParseArray(t *testing.T) {
	tests := []struct {
		name          string
		input         []byte
		expectedBytes int
		expectedDump  string
		expectError   bool
	}{
		{
			name:          "simple array",
			input:         []byte("*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n"),
			expectedBytes: 26,
			expectedDump:  "[hello,world]",
			expectError:   false,
		},
		{
			name:          "empty array",
			input:         []byte("*0\r\n"),
			expectedBytes: 4,
			expectedDump:  "[]",
			expectError:   false,
		},
		{
			name:          "nested types",
			input:         []byte("*3\r\n:1\r\n+OK\r\n$5\r\nhello\r\n"),
			expectedBytes: 24,
			expectedDump:  "[1,OK,hello]",
			expectError:   false,
		},
		{
			name:         "incomplete array",
			input:        []byte("*2\r\n$5\r\nhello\r\n"),
			expectedDump: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, bytesConsumed, err := parseArray(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Dump() != tt.expectedDump {
				t.Errorf("got dump %q, want %q", result.Dump(), tt.expectedDump)
			}

			if bytesConsumed != tt.expectedBytes {
				t.Errorf("got %d bytes consumed, want %d", bytesConsumed, tt.expectedBytes)
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       *RESPArray
		expected    RESPCommand
		expectError bool
	}{
		{
			name: "simple SET command",
			input: &RESPArray{
				data: []RESPData{
					&RESPBulkString{data: []byte("SET")},
					&RESPBulkString{data: []byte("key")},
					&RESPBulkString{data: []byte("value")},
				},
			},
			expected: RESPCommand{
				Type: SET,
				Args: []RESPData{
					&RESPBulkString{data: []byte("key")},
					&RESPBulkString{data: []byte("value")},
				},
			},
			expectError: false,
		},
		{
			name: "GET command with simple string",
			input: &RESPArray{
				data: []RESPData{
					&RESPSimpleString{data: "GET"},
					&RESPBulkString{data: []byte("mykey")},
				},
			},
			expected: RESPCommand{
				Type: GET,
				Args: []RESPData{
					&RESPBulkString{data: []byte("mykey")},
				},
			},
			expectError: false,
		},
		{
			name: "case insensitive command",
			input: &RESPArray{
				data: []RESPData{
					&RESPBulkString{data: []byte("echo")},
					&RESPBulkString{data: []byte("hello")},
				},
			},
			expected: RESPCommand{
				Type: ECHO,
				Args: []RESPData{
					&RESPBulkString{data: []byte("hello")},
				},
			},
			expectError: false,
		},
		{
			name: "unknown command",
			input: &RESPArray{
				data: []RESPData{
					&RESPBulkString{data: []byte("UNKNOWN")},
					&RESPBulkString{data: []byte("arg")},
				},
			},
			expectError: true,
		},
		{
			name: "invalid command type",
			input: &RESPArray{
				data: []RESPData{
					&RESPInteger{data: 123},
					&RESPBulkString{data: []byte("arg")},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCommand(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Type != tt.expected.Type {
				t.Errorf("got command type %v, want %v", result.Type, tt.expected.Type)
			}

			if !reflect.DeepEqual(result.Args, tt.expected.Args) {
				t.Errorf("got args %v, want %v", result.Args, tt.expected.Args)
			}
		})
	}
}

func TestReadCommands(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []RESPCommand
		expectError bool
	}{
		{
			name:  "single command",
			input: "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
			expected: []RESPCommand{
				{
					Type: SET,
					Args: []RESPData{
						&RESPBulkString{data: []byte("key")},
						&RESPBulkString{data: []byte("value")},
					},
				},
			},
			expectError: false,
		},
		{
			name: "multiple commands",
			input: "*2\r\n$3\r\nGET\r\n$3\r\nkey\r\n" +
				"*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			expected: []RESPCommand{
				{
					Type: GET,
					Args: []RESPData{
						&RESPBulkString{data: []byte("key")},
					},
				},
				{
					Type: SET,
					Args: []RESPData{
						&RESPBulkString{data: []byte("foo")},
						&RESPBulkString{data: []byte("bar")},
					},
				},
			},
			expectError: false,
		},
		{
			name:        "incomplete command",
			input:       "*2\r\n$3\r\nGET\r\n$3\r\nke",
			expected:    nil,
			expectError: false, // Should return no commands but no error
		},
		{
			name:        "invalid command format",
			input:       "*-1\r\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewRESPReader(bytes.NewReader([]byte(tt.input)), 1024)
			results, err := reader.ReadCommands()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil && err != io.EOF {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(results, tt.expected) {
				t.Errorf("got commands %v, want %v", results, tt.expected)
			}
		})
	}
}

func TestBufferBehavior(t *testing.T) {
	t.Run("buffer growth", func(t *testing.T) {
		// Create a reader with small initial buffer
		reader := NewRESPReader(bytes.NewReader([]byte{}), 2)
		originalSize := len(reader.buffer)

		// Write data larger than buffer
		largeCommand := "*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"
		reader.reader = bytes.NewReader([]byte(largeCommand))

		// Read commands should cause buffer to grow after initial read doesn't work
		reader.ReadCommands()
		reader.ReadCommands()
		reader.ReadCommands()

		if len(reader.buffer) <= originalSize {
			t.Errorf("buffer did not grow, size: %d, original size: %d", len(reader.buffer), originalSize)
		}
	})

	t.Run("partial command across reads", func(t *testing.T) {
		// Split a command across multiple reads
		part1 := "*3\r\n$3\r\nSET\r\n$3"
		part2 := "\r\nkey\r\n$5\r\nvalue\r\n"

		// Create a custom reader that returns the parts separately
		reader := NewRESPReader(&partialReader{parts: []string{part1, part2}}, 1024)

		// First read should return no commands (incomplete)
		commands, err := reader.ReadCommands()
		if err != nil && err != io.EOF {
			t.Fatalf("unexpected error on first read: %v", err)
		}
		if len(commands) != 0 {
			t.Errorf("expected no commands on first read, got %d", len(commands))
		}

		// Second read should complete the command
		commands, err = reader.ReadCommands()
		if err != nil && err != io.EOF {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		if len(commands) != 1 {
			t.Errorf("expected 1 command on second read, got %d", len(commands))
		}
	})
}

// Helper type for testing partial reads
type partialReader struct {
	parts    []string
	currPart int
}

func (r *partialReader) Read(p []byte) (n int, err error) {
	if r.currPart >= len(r.parts) {
		return 0, io.EOF
	}
	n = copy(p, r.parts[r.currPart])
	r.currPart++
	return n, nil
}
