package media

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jvzantvoort/bundle/bundle"
	"github.com/jvzantvoort/bundle/lock"
)

// testEntry is one TIFF tag: type 1 BYTE, 2 ASCII, 3 SHORT, 4 LONG, 5 RATIONAL.
type testEntry struct {
	tag, typ uint16
	count    uint32
	data     []byte
}

func tASCII(tag uint16, s string) testEntry {
	return testEntry{tag, 2, uint32(len(s) + 1), append([]byte(s), 0)}
}

func tShort(tag uint16, v uint16) testEntry {
	return testEntry{tag, 3, 1, binary.LittleEndian.AppendUint16(nil, v)}
}

func tLong(tag uint16, v uint32) testEntry {
	return testEntry{tag, 4, 1, binary.LittleEndian.AppendUint32(nil, v)}
}

func tRationals(tag uint16, pairs ...uint32) testEntry {
	var data []byte
	for _, v := range pairs {
		data = binary.LittleEndian.AppendUint32(data, v)
	}
	return testEntry{tag, 5, uint32(len(pairs) / 2), data}
}

func testIFDSize(entries []testEntry) uint32 {
	size := uint32(2 + 12*len(entries) + 4)
	for _, e := range entries {
		if len(e.data) > 4 {
			size += uint32(len(e.data)+1) &^ 1
		}
	}
	return size
}

func writeTestIFD(out *bytes.Buffer, offset uint32, entries []testEntry) {
	data := offset + uint32(2+12*len(entries)+4)
	var extra []byte
	out.Write(binary.LittleEndian.AppendUint16(nil, uint16(len(entries))))
	for _, e := range entries {
		var entry []byte
		entry = binary.LittleEndian.AppendUint16(entry, e.tag)
		entry = binary.LittleEndian.AppendUint16(entry, e.typ)
		entry = binary.LittleEndian.AppendUint32(entry, e.count)
		if len(e.data) <= 4 {
			entry = append(entry, append(e.data, make([]byte, 4-len(e.data))...)...)
		} else {
			entry = binary.LittleEndian.AppendUint32(entry, data+uint32(len(extra)))
			extra = append(extra, e.data...)
			if len(extra)%2 == 1 {
				extra = append(extra, 0)
			}
		}
		out.Write(entry)
	}
	out.Write([]byte{0, 0, 0, 0})
	out.Write(extra)
}

// exifTIFF builds a little-endian EXIF block with a camera, orientation 6
// (rotated 90 degrees), a capture time, a lens and a GPS position.
func exifTIFF() []byte {
	exifIFD := []testEntry{
		tRationals(0x829D, 28, 10), // FNumber 2.8
		tShort(0x8827, 100),        // ISOSpeedRatings
		tASCII(0x9003, "2024:06:01 10:20:30"),
		tASCII(0xA434, "RF24-105mm F4 L IS USM"),
	}
	gpsIFD := []testEntry{
		{0x0000, 1, 4, []byte{2, 3, 0, 0}}, // GPSVersionID
		tASCII(0x0001, "N"),
		tRationals(0x0002, 52, 1, 30, 1, 0, 1),
		tASCII(0x0003, "W"),
		tRationals(0x0004, 4, 1, 15, 1, 0, 1),
	}
	ifd0 := []testEntry{
		tASCII(0x010F, "Canon"),
		tASCII(0x0110, "Canon EOS R6\x00garbage"),
		tShort(0x0112, 6),
		tASCII(0x0131, "Firmware 1.8"),
		tLong(0x8769, 0),
		tLong(0x8825, 0),
	}
	exifOffset := 8 + testIFDSize(ifd0)
	gpsOffset := exifOffset + testIFDSize(exifIFD)
	ifd0[4] = tLong(0x8769, exifOffset)
	ifd0[5] = tLong(0x8825, gpsOffset)

	var out bytes.Buffer
	out.WriteString("II*\x00")
	out.Write(binary.LittleEndian.AppendUint32(nil, 8))
	writeTestIFD(&out, 8, ifd0)
	writeTestIFD(&out, exifOffset, exifIFD)
	writeTestIFD(&out, gpsOffset, gpsIFD)
	return out.Bytes()
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	for x := 0; x < 4; x++ {
		img.Set(x, 0, color.RGBA{R: 200, A: 255})
	}
	return img
}

// jpegWithEXIF encodes a 4x2 JPEG and inserts an APP1 EXIF segment.
func jpegWithEXIF(t *testing.T) []byte {
	t.Helper()
	var plain bytes.Buffer
	if err := jpeg.Encode(&plain, testImage(), nil); err != nil {
		t.Fatal(err)
	}
	payload := append([]byte("Exif\x00\x00"), exifTIFF()...)
	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	segment = append(segment, payload...)
	raw := plain.Bytes()
	return append(append(append([]byte{}, raw[:2]...), segment...), raw[2:]...)
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := png.Encode(&out, testImage()); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func write(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExtractImageWithEXIF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.jpg")
	write(t, path, jpegWithEXIF(t))

	info, err := Extract(path)
	if err != nil || info == nil {
		t.Fatalf("Extract = %v, %v", info, err)
	}
	if info.Type != "image" || info.Format != "JPEG" {
		t.Errorf("type/format = %q/%q", info.Type, info.Format)
	}
	// Orientation 6 turns the 4x2 image on its side
	if *info.Resolution.Width != 2 || *info.Resolution.Height != 4 {
		t.Errorf("resolution = %dx%d, want 2x4", *info.Resolution.Width, *info.Resolution.Height)
	}
	if info.TakenAt != "2024:06:01 10:20:30" {
		t.Errorf("taken_at = %q", info.TakenAt)
	}
	want := Device{Make: "Canon", Model: "Canon EOS R6", Lens: "RF24-105mm F4 L IS USM", Software: "Firmware 1.8"}
	if info.Device == nil || *info.Device != want {
		t.Errorf("device = %+v, want %+v", info.Device, want)
	}
	if info.GPS == nil || info.GPS.Latitude != 52.5 || info.GPS.Longitude != -4.25 {
		t.Errorf("gps = %+v", info.GPS)
	}
	if info.EXIF["FNumber"] != 2.8 || info.EXIF["ISOSpeedRatings"] != int64(100) {
		t.Errorf("exif FNumber/ISO = %v/%v", info.EXIF["FNumber"], info.EXIF["ISOSpeedRatings"])
	}
	if info.EXIF["GPSGPSLatitudeRef"] != "N" || info.EXIF["GPSGPSVersionID"] != "<4 bytes>" {
		t.Errorf("exif GPS = %v/%v", info.EXIF["GPSGPSLatitudeRef"], info.EXIF["GPSGPSVersionID"])
	}
	for _, skipped := range []string{"ExifIFDPointer", "GPSInfoIFDPointer"} {
		if _, ok := info.EXIF[skipped]; ok {
			t.Errorf("exif contains %s", skipped)
		}
	}
}

func TestExtractImageWithoutEXIFAndUnsupported(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "plain.png"), pngBytes(t))
	write(t, filepath.Join(dir, "notes.txt"), []byte("not an image"))
	write(t, filepath.Join(dir, "broken.jpg"), []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00})

	info, err := Extract(filepath.Join(dir, "plain.png"))
	if err != nil || info == nil {
		t.Fatalf("png: %v, %v", info, err)
	}
	if info.Format != "PNG" || *info.Resolution.Width != 4 || info.EXIF != nil || info.Device != nil {
		t.Errorf("png info = %+v", info)
	}
	for _, name := range []string{"notes.txt", "broken.jpg"} {
		if info, err := Extract(filepath.Join(dir, name)); info != nil || err != nil {
			t.Errorf("%s: Extract = %+v, %v, want unsupported", name, info, err)
		}
	}
	if _, err := Extract(filepath.Join(dir, "missing.png")); err == nil {
		t.Error("missing file: no error")
	}
}

func TestParseProbe(t *testing.T) {
	out := []byte(`{
	  "streams": [
	    {"codec_type": "audio", "codec_name": "aac"},
	    {"codec_type": "video", "codec_name": "hevc", "width": 1920, "height": 1080}
	  ],
	  "format": {"duration": "12.34567", "tags": {
	    "com.apple.quicktime.make": "Apple", "com.apple.quicktime.model": "iPhone 15",
	    "encoder": "Lavf", "creation_time": "2024-06-01T10:20:30.000000Z"}}
	}`)
	info, err := parseProbe(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Type != "video" || info.Codec != "hevc" || *info.Resolution.Width != 1920 || *info.DurationSeconds != 12.346 {
		t.Errorf("video = %+v", info)
	}
	if info.Device == nil || *info.Device != (Device{Make: "Apple", Model: "iPhone 15", Software: "Lavf"}) {
		t.Errorf("device = %+v", info.Device)
	}
	if info.TakenAt != "2024-06-01T10:20:30.000000Z" {
		t.Errorf("taken_at = %q", info.TakenAt)
	}
	if _, err := parseProbe([]byte("{")); err == nil {
		t.Error("bad ffprobe output: no error")
	}
}

func newBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "photos")
	write(t, filepath.Join(dir, "a.jpg"), jpegWithEXIF(t))
	write(t, filepath.Join(dir, "sub", "b.png"), pngBytes(t))
	write(t, filepath.Join(dir, "notes.txt"), []byte("hello"))
	b, err := bundle.Create(dir, "Photos")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestScanIsIncrementalAndKeepsIdentity(t *testing.T) {
	b := newBundle(t)
	opts := Options{Generator: "bundle test"}

	summary, err := Scan(b.Path, b.Files.Records, opts)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Added != 2 || summary.Unsupported != 1 || summary.Files != 2 || summary.Devices != 1 {
		t.Fatalf("first scan = %+v", summary)
	}
	doc, err := Load(b.Path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Generator != "bundle test" || doc.Version != SchemaVersion || doc.UpdatedAt == "" {
		t.Errorf("document header = %+v", doc)
	}
	if len(doc.Devices) != 1 || doc.Devices[0].Files != 1 || doc.Devices[0].Model != "Canon EOS R6" {
		t.Errorf("devices = %+v", doc.Devices)
	}
	if doc.Files["sub/b.png"].SHA256 == "" {
		t.Error("entries do not record the file checksum")
	}

	// Stored like bundle-python: sorted keys, two-space indent, no "path" field
	raw, _ := os.ReadFile(Path(b.Path))
	text := string(raw)
	if !strings.HasPrefix(text, "{\n  \"devices\": [") || strings.Index(text, `"files"`) > strings.Index(text, `"generator"`) {
		t.Errorf("MEDIA.json is not sorted:\n%s", text)
	}
	if strings.Contains(text, `"path"`) {
		t.Error("MEDIA.json entries contain a path field")
	}

	summary, err = Scan(b.Path, b.Files.Records, opts)
	if err != nil || summary.Skipped != 2 || summary.Added+summary.Updated != 0 {
		t.Fatalf("second scan = %+v, %v", summary, err)
	}
	summary, err = Scan(b.Path, b.Files.Records, Options{Force: true})
	if err != nil || summary.Updated != 2 {
		t.Fatalf("forced scan = %+v, %v", summary, err)
	}

	// MEDIA.json is outside the checksum: the bundle still verifies
	ok, corrupted, err := bundle.Verify(b.Path)
	if err != nil || !ok {
		t.Fatalf("verify after scan = %v %v %v", ok, corrupted, err)
	}
}

func TestScanRemovesStaleEntriesAndKeepsForeignFields(t *testing.T) {
	b := newBundle(t)
	// A file written by bundle-python, with an entry for a file that is gone
	sum := b.Files.Records[0].Checksum
	for _, r := range b.Files.Records {
		if r.FilePath == "a.jpg" {
			sum = r.Checksum
		}
	}
	write(t, Path(b.Path), []byte(`{"version": 1, "generator": "bundle-python 0.1.0",
	  "devices": [], "files": {
	    "gone.jpg": {"type": "image"},
	    "a.jpg": {"type": "image", "sha256": "`+sum+`", "exif": {"Custom": [1, 2]}}}}`))

	summary, err := Scan(b.Path, b.Files.Records, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Removed != 1 || summary.Skipped != 1 || summary.Added != 1 {
		t.Fatalf("scan = %+v", summary)
	}
	doc, _ := Load(b.Path)
	if _, ok := doc.Files["gone.jpg"]; ok {
		t.Error("stale entry kept")
	}
	if got, _ := json.Marshal(doc.Files["a.jpg"].EXIF); string(got) != `{"Custom":[1,2]}` {
		t.Errorf("unchanged entry was rewritten: %s", got)
	}
}

func TestScanRespectsLockAndBadFile(t *testing.T) {
	b := newBundle(t)
	held, err := lock.AcquireLock(b.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Scan(b.Path, b.Files.Records, Options{}); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Errorf("scan of a locked bundle: %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}

	write(t, Path(b.Path), []byte("{"))
	if _, err := Scan(b.Path, b.Files.Records, Options{}); err == nil {
		t.Error("scan with a corrupt MEDIA.json: no error")
	}
	if _, err := os.Stat(filepath.Join(b.Path, ".bundle", ".lock")); !os.IsNotExist(err) {
		t.Error("lock not released after a failed scan")
	}
}

func TestLoadMissingIsEmpty(t *testing.T) {
	doc, err := Load(t.TempDir())
	if err != nil || len(doc.Files) != 0 || doc.Devices == nil {
		t.Fatalf("Load of a bundle without MEDIA.json = %+v, %v", doc, err)
	}
}
