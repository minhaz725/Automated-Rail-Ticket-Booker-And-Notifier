// SPDX-License-Identifier: Unlicense OR BSD-3-Clause

package tables

import (
	"encoding/binary"
	"fmt"
)

// Code generated by binarygen from aat_ltag_src.go. DO NOT EDIT

func ParseLtag(src []byte) (Ltag, int, error) {
	var item Ltag
	n := 0
	if L := len(src); L < 12 {
		return item, 0, fmt.Errorf("reading Ltag: "+"EOF: expected length: 12, got %d", L)
	}
	_ = src[11] // early bound checking
	item.version = binary.BigEndian.Uint32(src[0:])
	item.flags = binary.BigEndian.Uint32(src[4:])
	item.numTags = binary.BigEndian.Uint32(src[8:])
	n += 12

	{
		arrayLength := int(item.numTags)

		if L := len(src); L < 12+arrayLength*4 {
			return item, 0, fmt.Errorf("reading Ltag: "+"EOF: expected length: %d, got %d", 12+arrayLength*4, L)
		}

		item.tagRange = make([]stringRange, arrayLength) // allocation guarded by the previous check
		for i := range item.tagRange {
			item.tagRange[i].mustParse(src[12+i*4:])
		}
		n += arrayLength * 4
	}
	{

		item.stringData = src[0:]
		n = len(src)
	}
	return item, n, nil
}

func (item *stringRange) mustParse(src []byte) {
	_ = src[3] // early bound checking
	item.offset = binary.BigEndian.Uint16(src[0:])
	item.length = binary.BigEndian.Uint16(src[2:])
}