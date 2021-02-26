package types

import (
	"bytes"
	"testing"
)

func TestNewBitArray(t *testing.T) {
	if NewBitArray(0) != nil {
		t.Error("bit array is not nil")
	}
}

func TestBitArraySize(t *testing.T) {
	b := NewBitArray(10)
	if b.Size() != 10 {
		t.Error("incorrect size of bit array")
	}

	b = NewBitArray(0)
	if b.Size() != 0 {
		t.Error("incorrect size of bit array")
	}
}

func TestBitArrayGetIndex(t *testing.T) {
	b := NewBitArray(0)
	if b.GetIndex(10) != false {
		t.Error("invalid index of bit array")
	}
}

func TestBitArraySetIndex(t *testing.T) {
	b := NewBitArray(0)
	if b.SetIndex(10, true) != false {
		t.Error("invalid index of bit array")
	}
}

func TestBitArrayBytes(t *testing.T) {
	b := NewBitArray(10)
	if !bytes.Equal(b.Bytes(), []byte{0, 0}) {
		t.Error("Bytes are not equal")
	}
}
