package media

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// A bounded EXIF reader. It reads IFD0, the Exif IFD and the GPS IFD of a
// TIFF structure (an EXIF block or a TIFF file) and names and shapes the
// values the way Pillow does, which bundle-python uses. It reads each IFD
// and value with ReadAt, within the data, so a crafted file cannot make it
// allocate much or loop.

// exifTag is a tag name and the number of values Pillow expects for it:
// -1 when unspecified, 0 when variable.
type exifTag struct {
	name   string
	length int
}

const (
	tagExifIFD     = 0x8769
	tagGPSIFD      = 0x8825
	tagInteropIFD  = 0xA005
	tagMakerNote   = 0x927C
	tagUserComment = 0x9286

	// maxValueBytes and maxValues bound one tag; larger tags are skipped.
	maxValueBytes = 1 << 20
	maxValues     = 1 << 16
)

// TIFF field types and the size of one value.
const (
	typeByte      = 1
	typeASCII     = 2
	typeShort     = 3
	typeLong      = 4
	typeRational  = 5
	typeSByte     = 6
	typeUndefined = 7
	typeSShort    = 8
	typeSLong     = 9
	typeSRational = 10
	typeFloat     = 11
	typeDouble    = 12
	typeIFD       = 13
)

var typeSizes = map[uint16]uint64{
	typeByte: 1, typeASCII: 1, typeShort: 2, typeLong: 4, typeRational: 8,
	typeSByte: 1, typeUndefined: 1, typeSShort: 2, typeSLong: 4, typeSRational: 8,
	typeFloat: 4, typeDouble: 8, typeIFD: 4,
}

type ifdEntry struct {
	tag, typ uint16
	count    uint64
	value    []byte
}

type exifReader struct {
	r     io.ReaderAt
	size  int64
	order binary.ByteOrder
	seen  map[uint64]bool
}

func (x *exifReader) read(offset, n uint64) ([]byte, bool) {
	if n > maxValueBytes || offset > uint64(x.size) || n > uint64(x.size)-offset {
		return nil, false
	}
	buf := make([]byte, n)
	if _, err := x.r.ReadAt(buf, int64(offset)); err != nil {
		return nil, false
	}
	return buf, true
}

// parseEXIF returns the EXIF tags of a TIFF structure by name, or nil when
// it has none. GPS tags get a "GPS" prefix on Pillow's GPS name, as in
// bundle-python ("GPSGPSLatitude").
func parseEXIF(r io.ReaderAt, size int64) map[string]any {
	x := &exifReader{r: r, size: size, seen: map[uint64]bool{}}
	head, ok := x.read(0, 8)
	if !ok {
		return nil
	}
	switch string(head[0:4]) {
	case "II*\x00":
		x.order = binary.LittleEndian
	case "MM\x00*":
		x.order = binary.BigEndian
	default:
		return nil
	}
	ifd0 := x.ifd(uint64(x.order.Uint32(head[4:8])))
	if ifd0 == nil {
		return nil
	}
	tags := map[string]any{}
	x.collect(tags, ifd0, mainTags, "")
	if offset, ok := x.pointer(ifd0, tagExifIFD); ok {
		x.collect(tags, x.ifd(offset), exifTags, "")
	}
	if offset, ok := x.pointer(ifd0, tagGPSIFD); ok {
		x.collect(tags, x.ifd(offset), gpsTags, "GPS")
	}
	if len(tags) == 0 {
		return nil
	}
	return tags
}

// ifd reads the entries of the IFD at offset, or nil when it is out of the
// data or was read before (a loop).
func (x *exifReader) ifd(offset uint64) []ifdEntry {
	if x.seen[offset] {
		return nil
	}
	x.seen[offset] = true
	raw, ok := x.read(offset, 2)
	if !ok {
		return nil
	}
	n := uint64(x.order.Uint16(raw))
	raw, ok = x.read(offset+2, 12*n)
	if !ok {
		return nil
	}
	entries := make([]ifdEntry, 0, n)
	for i := uint64(0); i < n; i++ {
		field := raw[12*i : 12*i+12]
		entry := ifdEntry{
			tag:   x.order.Uint16(field[0:]),
			typ:   x.order.Uint16(field[2:]),
			count: uint64(x.order.Uint32(field[4:])),
		}
		size, known := typeSizes[entry.typ]
		if !known || entry.count == 0 {
			continue
		}
		total := size * entry.count // at most 8 * (2^32-1): no overflow in uint64
		if total <= 4 {
			entry.value = field[8 : 8+total]
		} else if value, ok := x.read(uint64(x.order.Uint32(field[8:])), total); ok {
			entry.value = value
		} else {
			continue // value outside the data or too large: skipped, like Pillow does
		}
		entries = append(entries, entry)
	}
	return entries
}

func (x *exifReader) pointer(entries []ifdEntry, tag uint16) (uint64, bool) {
	for _, e := range entries {
		if e.tag == tag && (e.typ == typeLong || e.typ == typeIFD) {
			return uint64(x.order.Uint32(e.value)), true
		}
	}
	return 0, false
}

func (x *exifReader) collect(tags map[string]any, entries []ifdEntry, names map[uint16]exifTag, prefix string) {
	for _, e := range entries {
		switch e.tag {
		case tagExifIFD, tagGPSIFD, tagInteropIFD, tagMakerNote, tagUserComment:
			continue
		}
		info, known := names[e.tag]
		if !known {
			info = exifTag{name: fmt.Sprintf("Tag%d", e.tag), length: -1}
		}
		if value, ok := x.value(e, info.length); ok {
			tags[prefix+info.name] = value
		}
	}
}

// value converts an entry the way Pillow does: BYTE, UNDEFINED and ASCII data
// is one value; a tag specified as single-valued keeps its first value; one
// value is a scalar and several are a list.
func (x *exifReader) value(e ifdEntry, length int) (any, bool) {
	switch e.typ {
	case typeByte, typeUndefined:
		return bytesValue(e.value), true
	case typeASCII:
		text := e.value
		if len(text) > 0 && text[len(text)-1] == 0 {
			text = text[:len(text)-1]
		}
		runes := make([]rune, len(text)) // latin-1
		for i, b := range text {
			runes[i] = rune(b)
		}
		return string(runes), true
	}
	if e.count > maxValues {
		return nil, false
	}
	size := typeSizes[e.typ]
	values := make([]any, e.count)
	for i := range values {
		v := e.value[uint64(i)*size:]
		switch e.typ {
		case typeShort:
			values[i] = int64(x.order.Uint16(v))
		case typeLong, typeIFD:
			values[i] = int64(x.order.Uint32(v))
		case typeSByte:
			values[i] = int64(int8(v[0]))
		case typeSShort:
			values[i] = int64(int16(x.order.Uint16(v)))
		case typeSLong:
			values[i] = int64(int32(x.order.Uint32(v)))
		case typeRational:
			values[i] = ratio(float64(x.order.Uint32(v)), float64(x.order.Uint32(v[4:])))
		case typeSRational:
			values[i] = ratio(float64(int32(x.order.Uint32(v))), float64(int32(x.order.Uint32(v[4:]))))
		case typeFloat:
			values[i] = finite(float64(math.Float32frombits(x.order.Uint32(v))))
		case typeDouble:
			values[i] = finite(math.Float64frombits(x.order.Uint64(v)))
		}
	}
	if length == 1 || len(values) == 1 {
		return values[0], true
	}
	return values, true
}

// ratio is num/den, or nil for a zero denominator (NaN in Pillow, which is
// not valid JSON).
func ratio(num, den float64) any {
	if den == 0 {
		return nil
	}
	return num / den
}

func finite(v float64) any {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return v
}
