//go:build test
// +build test

package miniredis

import (
	"bytes"
	"testing"
)

func TestStringData_Type(t *testing.T) {
	tests := []struct {
		name string
		data StringData
		want MiniRedisDataType
	}{
		{
			name: "empty string",
			data: StringData{data: []byte("")},
			want: Scalar,
		},
		{
			name: "non-empty string",
			data: StringData{data: []byte("hello")},
			want: Scalar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.Type(); got != tt.want {
				t.Errorf("StringData.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegerData_Type(t *testing.T) {
	tests := []struct {
		name string
		data IntegerData
		want MiniRedisDataType
	}{
		{
			name: "zero",
			data: IntegerData{data: 0},
			want: Scalar,
		},
		{
			name: "positive integer",
			data: IntegerData{data: 42},
			want: Scalar,
		},
		{
			name: "negative integer",
			data: IntegerData{data: -42},
			want: Scalar,
		},
		{
			name: "max int64",
			data: IntegerData{data: 9223372036854775807},
			want: Scalar,
		},
		{
			name: "min int64",
			data: IntegerData{data: -9223372036854775808},
			want: Scalar,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.data.Type(); got != tt.want {
				t.Errorf("IntegerData.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringData_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		data    StringData
		want    []byte
		wantErr bool
	}{
		{
			name: "empty string",
			data: StringData{data: []byte("")},
			want: []byte("$0\r\n\r\n"),
		},
		{
			name: "simple string",
			data: StringData{data: []byte("hello")},
			want: []byte("$5\r\nhello\r\n"),
		},
		{
			name: "string with spaces",
			data: StringData{data: []byte("hello world")},
			want: []byte("$11\r\nhello world\r\n"),
		},
		{
			name: "string with special characters",
			data: StringData{data: []byte("hello\nworld")},
			want: []byte("$11\r\nhello\nworld\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.data.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("StringData.Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("StringData.Serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntegerData_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		data    IntegerData
		want    []byte
		wantErr bool
	}{
		{
			name: "zero",
			data: IntegerData{data: 0},
			want: []byte(":0\r\n"),
		},
		{
			name: "positive integer",
			data: IntegerData{data: 42},
			want: []byte(":42\r\n"),
		},
		{
			name: "negative integer",
			data: IntegerData{data: -42},
			want: []byte(":-42\r\n"),
		},
		{
			name: "max int64",
			data: IntegerData{data: 9223372036854775807},
			want: []byte(":9223372036854775807\r\n"),
		},
		{
			name: "min int64",
			data: IntegerData{data: -9223372036854775808},
			want: []byte(":-9223372036854775808\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.data.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("IntegerData.Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("IntegerData.Serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}
