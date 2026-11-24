package upload

import (
	"bytes"
	"encoding/binary"
)

// jpegOrientation returns the EXIF orientation (1 to 8) of a JPEG, or 1 when absent or unreadable.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 1
		}
		marker := data[pos+1]
		if marker == 0xFF {
			pos++
			continue
		}
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[pos+2 : pos+4]))
		end := pos + 2 + size
		if size < 2 || end > len(data) {
			return 1
		}
		if marker == 0xE1 {
			if o := tiffOrientation(data[pos+4 : end]); o != 0 {
				return o
			}
		}
		pos = end
	}
	return 1
}

func tiffOrientation(seg []byte) int {
	if !bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
		return 0
	}
	t := seg[6:]
	if len(t) < 8 {
		return 0
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}
	ifd := int(bo.Uint32(t[4:8]))
	if ifd < 8 || ifd+2 > len(t) {
		return 0
	}
	n := int(bo.Uint16(t[ifd : ifd+2]))
	for i := 0; i < n; i++ {
		e := ifd + 2 + i*12
		if e+12 > len(t) {
			return 0
		}
		if bo.Uint16(t[e:e+2]) == 0x0112 {
			if v := int(bo.Uint16(t[e+8 : e+10])); v >= 1 && v <= 8 {
				return v
			}
			return 0
		}
	}
	return 0
}
