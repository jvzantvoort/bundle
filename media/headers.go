package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// Bounded readers for the parts of image files this package needs: they
// only read fixed-size headers or one JPEG segment at a time, so a crafted
// file cannot make them allocate much or loop.

var errNoEXIF = errors.New("no EXIF data")

// jpegEXIF returns the EXIF block (a TIFF structure) of a JPEG file, or
// errNoEXIF. It walks the marker segments up to the image data and reads
// one segment at a time, each at most 64 KiB.
func jpegEXIF(r io.Reader) ([]byte, error) {
	marker := make([]byte, 2)
	if _, err := io.ReadFull(r, marker); err != nil || marker[0] != 0xFF || marker[1] != 0xD8 {
		return nil, errNoEXIF
	}
	for {
		if _, err := io.ReadFull(r, marker); err != nil || marker[0] != 0xFF {
			return nil, errNoEXIF
		}
		for marker[1] == 0xFF { // fill bytes
			if _, err := io.ReadFull(r, marker[1:]); err != nil {
				return nil, errNoEXIF
			}
		}
		switch {
		case marker[1] == 0xDA || marker[1] == 0xD9: // start of scan, end of image
			return nil, errNoEXIF
		case marker[1] >= 0xD0 && marker[1] <= 0xD7 || marker[1] == 0x01: // no length
			continue
		}
		var length uint16
		if err := binary.Read(r, binary.BigEndian, &length); err != nil || length < 2 {
			return nil, errNoEXIF
		}
		segment := make([]byte, length-2)
		if _, err := io.ReadFull(r, segment); err != nil {
			return nil, errNoEXIF
		}
		if marker[1] == 0xE1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return segment[6:], nil
		}
	}
}

// webpSize reads the dimensions from the header of a WebP file.
func webpSize(header []byte) (int, int, bool) {
	if len(header) < 30 || string(header[0:4]) != "RIFF" || string(header[8:12]) != "WEBP" {
		return 0, 0, false
	}
	switch string(header[12:16]) {
	case "VP8 ": // lossy: frame tag (3) and start code (3) precede the sizes
		if header[23] != 0x9D || header[24] != 0x01 || header[25] != 0x2A {
			return 0, 0, false
		}
		w := int(binary.LittleEndian.Uint16(header[26:]) & 0x3FFF)
		h := int(binary.LittleEndian.Uint16(header[28:]) & 0x3FFF)
		return w, h, w > 0 && h > 0
	case "VP8L": // lossless: signature byte, then 14-bit width-1 and height-1
		if header[20] != 0x2F {
			return 0, 0, false
		}
		bits := binary.LittleEndian.Uint32(header[21:])
		return int(bits&0x3FFF) + 1, int(bits>>14&0x3FFF) + 1, true
	case "VP8X": // extended: 24-bit canvas width-1 and height-1
		w := int(header[24]) | int(header[25])<<8 | int(header[26])<<16
		h := int(header[27]) | int(header[28])<<8 | int(header[29])<<16
		return w + 1, h + 1, true
	}
	return 0, 0, false
}

// bmpSize reads the dimensions from the header of a BMP file.
func bmpSize(header []byte) (int, int, bool) {
	if len(header) < 26 || string(header[0:2]) != "BM" {
		return 0, 0, false
	}
	if binary.LittleEndian.Uint32(header[14:]) == 12 { // OS/2 BITMAPCOREHEADER
		w, h := int(binary.LittleEndian.Uint16(header[18:])), int(binary.LittleEndian.Uint16(header[20:]))
		return w, h, w > 0 && h > 0
	}
	w := int(int32(binary.LittleEndian.Uint32(header[18:])))
	h := int(int32(binary.LittleEndian.Uint32(header[22:])))
	if h < 0 { // top-down bitmap
		h = -h
	}
	return w, h, w > 0 && h > 0
}
