// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package hexutil

import (
	"bytes"
	"math/big"
	"testing"
)

type marshalTest struct {
	input interface{}
	want  string
}

type unmarshalTest struct {
	input        string
	want         interface{}
	wantErr      error // if set, decoding must fail on any platform
	wantErr32bit error // if set, decoding must fail on 32bit platforms (used for Uint tests)
}

var (
	encodeBytesTests = []marshalTest{
		{[]byte{}, "Mx"},
		{[]byte{0}, "Mx00"},
		{[]byte{0, 0, 1, 2}, "Mx00000102"},
	}

	encodeBigTests = []marshalTest{
		{referenceBig("0"), "Mx0"},
		{referenceBig("1"), "Mx1"},
		{referenceBig("ff"), "Mxff"},
		{referenceBig("112233445566778899aabbccddeeff"), "Mx112233445566778899aabbccddeeff"},
		{referenceBig("80a7f2c1bcc396c00"), "Mx80a7f2c1bcc396c00"},
		{referenceBig("-80a7f2c1bcc396c00"), "Mx-80a7f2c1bcc396c00"},
	}

	encodeUint64Tests = []marshalTest{
		{uint64(0), "Mx0"},
		{uint64(1), "Mx1"},
		{uint64(0xff), "Mxff"},
		{uint64(0x1122334455667788), "Mx1122334455667788"},
	}

	encodeUintTests = []marshalTest{
		{uint(0), "Mx0"},
		{uint(1), "Mx1"},
		{uint(0xff), "Mxff"},
		{uint(0x11223344), "Mx11223344"},
	}

	decodeBytesTests = []unmarshalTest{
		// invalid
		{input: ``, wantErr: ErrEmptyString},
		{input: `0`, wantErr: ErrMissingPrefix},
		{input: `Mx0`, wantErr: ErrOddLength},
		{input: `Mx023`, wantErr: ErrOddLength},
		{input: `Mxxx`, wantErr: ErrSyntax},
		{input: `Mx01zz01`, wantErr: ErrSyntax},
		// valid
		{input: `Mx`, want: []byte{}},
		{input: `MX`, want: []byte{}},
		{input: `Mx02`, want: []byte{0x02}},
		{input: `MX02`, want: []byte{0x02}},
		{input: `Mxffffffffff`, want: []byte{0xff, 0xff, 0xff, 0xff, 0xff}},
		{
			input: `Mxffffffffffffffffffffffffffffffffffff`,
			want:  []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}

	decodeBigTests = []unmarshalTest{
		// invalid
		{input: `0`, wantErr: ErrMissingPrefix},
		{input: `Mx`, wantErr: ErrEmptyNumber},
		{input: `Mx01`, wantErr: ErrLeadingZero},
		{input: `Mxx`, wantErr: ErrSyntax},
		{input: `Mx1zz01`, wantErr: ErrSyntax},
		{
			input:   `Mx10000000000000000000000000000000000000000000000000000000000000000`,
			wantErr: ErrBig256Range,
		},
		// valid
		{input: `Mx0`, want: big.NewInt(0)},
		{input: `Mx2`, want: big.NewInt(0x2)},
		{input: `Mx2F2`, want: big.NewInt(0x2f2)},
		{input: `MX2F2`, want: big.NewInt(0x2f2)},
		{input: `Mx1122aaff`, want: big.NewInt(0x1122aaff)},
		{input: `MxbBb`, want: big.NewInt(0xbbb)},
		{input: `Mxfffffffff`, want: big.NewInt(0xfffffffff)},
		{
			input: `Mx112233445566778899aabbccddeeff`,
			want:  referenceBig("112233445566778899aabbccddeeff"),
		},
		{
			input: `Mxffffffffffffffffffffffffffffffffffff`,
			want:  referenceBig("ffffffffffffffffffffffffffffffffffff"),
		},
		{
			input: `Mxffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff`,
			want:  referenceBig("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		},
	}

	decodeUint64Tests = []unmarshalTest{
		// invalid
		{input: `0`, wantErr: ErrMissingPrefix},
		{input: `Mx`, wantErr: ErrEmptyNumber},
		{input: `Mx01`, wantErr: ErrLeadingZero},
		{input: `Mxfffffffffffffffff`, wantErr: ErrUint64Range},
		{input: `Mxx`, wantErr: ErrSyntax},
		{input: `Mx1zz01`, wantErr: ErrSyntax},
		// valid
		{input: `Mx0`, want: uint64(0)},
		{input: `Mx2`, want: uint64(0x2)},
		{input: `Mx2F2`, want: uint64(0x2f2)},
		{input: `MX2F2`, want: uint64(0x2f2)},
		{input: `Mx1122aaff`, want: uint64(0x1122aaff)},
		{input: `Mxbbb`, want: uint64(0xbbb)},
		{input: `Mxffffffffffffffff`, want: uint64(0xffffffffffffffff)},
	}
)

func TestEncode(t *testing.T) {
	for _, test := range encodeBytesTests {
		enc := Encode(test.input.([]byte))
		if enc != test.want {
			t.Errorf("input %x: wrong encoding %s", test.input, enc)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, test := range decodeBytesTests {
		dec, err := Decode(test.input)
		if !checkError(t, test.input, err, test.wantErr) {
			continue
		}
		if !bytes.Equal(test.want.([]byte), dec) {
			t.Errorf("input %s: value mismatch: got %x, want %x", test.input, dec, test.want)
			continue
		}
	}
}

func TestEncodeBig(t *testing.T) {
	for _, test := range encodeBigTests {
		enc := EncodeBig(test.input.(*big.Int))
		if enc != test.want {
			t.Errorf("input %x: wrong encoding %s", test.input, enc)
		}
	}
}

func TestDecodeBig(t *testing.T) {
	for _, test := range decodeBigTests {
		dec, err := DecodeBig(test.input)
		if !checkError(t, test.input, err, test.wantErr) {
			continue
		}
		if dec.Cmp(test.want.(*big.Int)) != 0 {
			t.Errorf("input %s: value mismatch: got %x, want %x", test.input, dec, test.want)
			continue
		}
	}
}

func TestEncodeUint64(t *testing.T) {
	for _, test := range encodeUint64Tests {
		enc := EncodeUint64(test.input.(uint64))
		if enc != test.want {
			t.Errorf("input %x: wrong encoding %s", test.input, enc)
		}
	}
}

func TestDecodeUint64(t *testing.T) {
	for _, test := range decodeUint64Tests {
		dec, err := DecodeUint64(test.input)
		if !checkError(t, test.input, err, test.wantErr) {
			continue
		}
		if dec != test.want.(uint64) {
			t.Errorf("input %s: value mismatch: got %x, want %x", test.input, dec, test.want)
			continue
		}
	}
}
