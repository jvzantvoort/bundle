package media

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"testing"
	"time"
)

func le16(v uint16) []byte { return binary.LittleEndian.AppendUint16(nil, v) }
func le32(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }

func riff(chunk string, payload []byte) []byte {
	out := append([]byte("RIFF"), le32(uint32(12+len(payload)))...)
	out = append(out, "WEBP"+chunk...)
	out = append(out, le32(uint32(len(payload)))...)
	return append(out, payload...)
}

func TestHeaderSizes(t *testing.T) {
	dir := t.TempDir()
	vp8 := append([]byte{0x10, 0x02, 0x00, 0x9D, 0x01, 0x2A}, append(le16(640), le16(480)...)...)
	vp8l := append([]byte{0x2F}, le32((7-1)|(5-1)<<14)...)
	vp8x := append(make([]byte, 4), 0xFF, 0x0F, 0x00, 0x37, 0x0B, 0x00) // 4096 x 2872
	bmp := append([]byte("BM"), make([]byte, 12)...)
	bmp = append(bmp, le32(40)...)
	bmp = append(bmp, le32(9)...)
	bmp = append(bmp, le32(uint32(0xFFFFFFFD))...) // height -3: top-down
	bmp = append(bmp, make([]byte, 16)...)

	// A TIFF file: its size is in the ImageWidth and ImageLength tags
	var tiffFile bytes.Buffer
	tiffFile.WriteString("II*\x00")
	tiffFile.Write(le32(8))
	writeTestIFD(&tiffFile, 8, []testEntry{tShort(0x0100, 320), tShort(0x0101, 200), tASCII(0x010F, "Nikon")})

	cases := map[string]struct {
		data          []byte
		format        string
		width, height int
	}{
		"lossy.webp":    {riff("VP8 ", append(vp8, make([]byte, 8)...)), "WEBP", 640, 480},
		"lossless.webp": {riff("VP8L", append(vp8l, make([]byte, 8)...)), "WEBP", 7, 5},
		"extended.webp": {riff("VP8X", append(vp8x, make([]byte, 8)...)), "WEBP", 4096, 2872},
		"top-down.bmp":  {bmp, "BMP", 9, 3},
		"scan.tif":      {tiffFile.Bytes(), "TIFF", 320, 200},
	}
	for name, want := range cases {
		path := filepath.Join(dir, name)
		write(t, path, want.data)
		info, err := Extract(path)
		if err != nil || info == nil {
			t.Errorf("%s: Extract = %v, %v", name, info, err)
			continue
		}
		if info.Format != want.format || *info.Resolution.Width != want.width || *info.Resolution.Height != want.height {
			t.Errorf("%s: %s %dx%d, want %s %dx%d", name, info.Format,
				*info.Resolution.Width, *info.Resolution.Height, want.format, want.width, want.height)
		}
	}
	if info, _ := Extract(filepath.Join(dir, "scan.tif")); info == nil || info.Device == nil || info.Device.Make != "Nikon" {
		t.Errorf("TIFF EXIF not read: %+v", info)
	}
}

// loopingEXIF has two main IFDs that point at each other.
func loopingEXIF() []byte {
	var out bytes.Buffer
	out.WriteString("II*\x00")
	out.Write(le32(8))
	out.Write(le16(0)) // IFD at 8: no entries, next at 14
	out.Write(le32(14))
	out.Write(le16(0)) // IFD at 14: no entries, next at 8
	out.Write(le32(8))
	return out.Bytes()
}

func TestLoopingIFDChainIsHarmless(t *testing.T) {
	if tags := parseEXIF(bytes.NewReader(loopingEXIF()), int64(len(loopingEXIF()))); tags != nil {
		t.Fatalf("looping IFD chain gave tags: %v", tags)
	}

	// The image is still read, without its EXIF data
	jpg := jpegWithEXIF(t)
	payload := append([]byte("Exif\x00\x00"), loopingEXIF()...)
	segment := append([]byte{0xFF, 0xE1}, binary.BigEndian.AppendUint16(nil, uint16(len(payload)+2))...)
	looping := append(append(append([]byte{}, jpg[:2]...), append(segment, payload...)...), jpg[2:]...)
	path := filepath.Join(t.TempDir(), "loop.jpg")
	write(t, path, looping)

	done := make(chan *File, 1)
	go func() { info, _ := Extract(path); done <- info }()
	select {
	case info := <-done:
		if info == nil || info.Resolution == nil || info.EXIF != nil {
			t.Errorf("looping EXIF: %+v", info)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Extract does not finish on a looping IFD chain")
	}
}

// FuzzReadEXIF feeds arbitrary bytes through the EXIF path: it must not
// panic, hang or allocate without bound.
func FuzzReadEXIF(f *testing.F) {
	f.Add(exifTIFF())
	f.Add(loopingEXIF())
	f.Add(append([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x10}, "Exif\x00\x00II*\x00\x08\x00\x00\x00"...))
	f.Fuzz(func(t *testing.T, data []byte) {
		_ = parseEXIF(bytes.NewReader(data), int64(len(data)))
		if block, err := jpegEXIF(bytes.NewReader(data)); err == nil {
			_ = parseEXIF(bytes.NewReader(block), int64(len(block)))
		}
		webpSize(data)
		bmpSize(data)
	})
}
