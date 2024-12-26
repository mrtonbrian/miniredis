package miniredis

import (
	"bytes"
	"strconv"
	"time"
)

type MiniRedisDataType int16

const (
	Scalar MiniRedisDataType = iota
	List
)

type MiniRedisObject struct {
	data   MiniRedisData
	expiry time.Time
}

type MiniRedisData interface {
	Type() MiniRedisDataType
	Serialize() ([]byte, error)
}

type StringData struct{ data []byte }
type IntegerData struct{ data int64 }

func (s *StringData) Type() MiniRedisDataType  { return Scalar }
func (s *IntegerData) Type() MiniRedisDataType { return Scalar }

func (s *StringData) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("$")
	buffer.WriteString(strconv.Itoa(len(s.data)))
	buffer.WriteString("\r\n")
	buffer.Write(s.data)
	buffer.WriteString("\r\n")
	return buffer.Bytes(), nil
}

func (n *IntegerData) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(":")
	buffer.WriteString(strconv.FormatInt(n.data, 10))
	buffer.WriteString("\r\n")
	return buffer.Bytes(), nil
}
