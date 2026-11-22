package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"  // register decoders for image.DecodeConfig; WebP, BMP
	_ "image/jpeg" // and TIFF sizes are read from their headers instead,
	_ "image/png"  // so no full decoder runs on them
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
)

// videoExtensions are read with ffprobe; anything else is tried as an image.
var videoExtensions = map[string]bool{
	".mp4": true, ".mov": true, ".m4v": true, ".mkv": true,
	".avi": true, ".webm": true, ".mts": true, ".3gp": true,
}

// orientationSwapsAxes are the EXIF orientations that rotate by 90 degrees.
var orientationSwapsAxes = map[int64]bool{5: true, 6: true, 7: true, 8: true}

// ffprobeTimeout bounds how long one video may be probed.
const ffprobeTimeout = 60 * time.Second

// extractor reads media files and warns once per scan about missing tools.
type extractor struct {
	warnedNoFFprobe   bool
	warnedProbeFailed bool
}

// Extract returns the media metadata of one file, or nil when it is not an
// image or video this package can read.
//
// Parameters:
//   - path: the file to read
//
// Returns:
//   - *File: the metadata (SHA256 is not set), or nil for unsupported files
//   - error: if the file cannot be opened or read
func Extract(path string) (*File, error) {
	return (&extractor{}).extract(path)
}

func (e *extractor) extract(path string) (*File, error) {
	if videoExtensions[strings.ToLower(filepath.Ext(path))] {
		return e.video(path)
	}
	return readImage(path)
}

func readImage(path string) (*File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	header := make([]byte, 32)
	n, err := io.ReadFull(f, header)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	header = header[:n]
	rewind := func() error { _, err := f.Seek(0, io.SeekStart); return err }
	if err := rewind(); err != nil {
		return nil, err
	}

	var width, height int
	var format string
	isTIFF := bytes.HasPrefix(header, []byte("II*\x00")) || bytes.HasPrefix(header, []byte("MM\x00*"))
	if w, h, ok := webpSize(header); ok {
		width, height, format = w, h, "WEBP"
	} else if w, h, ok := bmpSize(header); ok {
		width, height, format = w, h, "BMP"
	} else if isTIFF {
		format = "TIFF"
	} else {
		config, name, err := image.DecodeConfig(f)
		if err != nil {
			log.Debugf("%s is not an image this tool reads: %v", path, err)
			return nil, nil
		}
		width, height, format = config.Width, config.Height, strings.ToUpper(name)
		if err := rewind(); err != nil {
			return nil, err
		}
	}

	tags := readEXIF(f, isTIFF)
	if isTIFF {
		w, okW := tags["ImageWidth"].(int64)
		h, okH := tags["ImageLength"].(int64)
		if !okW || !okH {
			log.Debugf("%s: no image size in its TIFF tags", path)
			return nil, nil
		}
		width, height = int(w), int(h)
	}
	info := &File{Type: "image", Format: format}
	if orientation, ok := tags["Orientation"].(int64); ok && orientationSwapsAxes[orientation] {
		width, height = height, width
	}
	info.Resolution = &Resolution{Width: &width, Height: &height}
	if len(tags) == 0 {
		return info, nil
	}
	info.EXIF = tags
	for _, name := range []string{"DateTimeOriginal", "DateTime"} {
		if taken, ok := tags[name].(string); ok && taken != "" {
			info.TakenAt = taken
			break
		}
	}
	info.Device = deviceFromEXIF(tags)
	info.GPS = gpsFromEXIF(tags)
	return info, nil
}

// readEXIF returns the EXIF tags of a JPEG or TIFF file by name, or nil.
func readEXIF(f *os.File, isTIFF bool) map[string]any {
	if isTIFF {
		info, err := f.Stat()
		if err != nil {
			return nil
		}
		return parseEXIF(f, info.Size())
	}
	block, err := jpegEXIF(f)
	if err != nil {
		return nil
	}
	return parseEXIF(bytes.NewReader(block), int64(len(block)))
}

// bytesValue renders BYTE and UNDEFINED data like bundle-python: as text
// when it is printable after trimming NULs and spaces, else as "<N bytes>".
func bytesValue(raw []byte) string {
	text := strings.Trim(strings.ToValidUTF8(string(raw), string(utf8.RuneError)), "\x00 ")
	for _, r := range text {
		if !unicode.IsPrint(r) {
			return fmt.Sprintf("<%d bytes>", len(raw))
		}
	}
	return text
}

// cleanText drops the NUL padding and garbage some cameras put after a value.
func cleanText(value any) string {
	text, _ := value.(string)
	text = strings.TrimSpace(strings.SplitN(text, "\x00", 2)[0])
	for _, r := range text {
		if !unicode.IsPrint(r) {
			return ""
		}
	}
	return text
}

func deviceFromEXIF(tags map[string]any) *Device {
	device := Device{
		Make:     cleanText(tags["Make"]),
		Model:    cleanText(tags["Model"]),
		Lens:     cleanText(tags["LensModel"]),
		Software: cleanText(tags["Software"]),
	}
	if device == (Device{}) {
		return nil
	}
	return &device
}

func gpsFromEXIF(tags map[string]any) *GPS {
	lat, okLat := degrees(tags["GPSGPSLatitude"], tags["GPSGPSLatitudeRef"])
	lon, okLon := degrees(tags["GPSGPSLongitude"], tags["GPSGPSLongitudeRef"])
	if !okLat || !okLon {
		return nil
	}
	return &GPS{Latitude: round6(lat), Longitude: round6(lon)}
}

func degrees(value, ref any) (float64, bool) {
	parts, ok := value.([]any)
	if !ok || len(parts) != 3 {
		return 0, false
	}
	var dms [3]float64
	for i, part := range parts {
		f, ok := part.(float64)
		if !ok {
			return 0, false
		}
		dms[i] = f
	}
	result := dms[0] + dms[1]/60 + dms[2]/3600
	if r, _ := ref.(string); r == "S" || r == "W" {
		result = -result
	}
	return result, true
}

func round6(v float64) float64 { return math.Round(v*1e6) / 1e6 }

func (e *extractor) video(path string) (*File, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		log.Debugf("ffprobe not found, skipping video %s", path)
		if !e.warnedNoFFprobe {
			e.warnedNoFFprobe = true
			log.Warn("ffprobe not found: video files are skipped (install ffmpeg)")
		}
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), ffprobeTimeout)
	defer cancel()
	var stderr strings.Builder
	cmd := exec.CommandContext(ctx, ffprobe, "-v", "error", "-print_format", "json", "-show_format", "-show_streams", path)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		log.Debugf("ffprobe failed for %s: %v: %s", path, err, strings.TrimSpace(stderr.String()))
		if !e.warnedProbeFailed {
			e.warnedProbeFailed = true
			log.Warn("ffprobe failed on some videos; use -v to see which. An ffmpeg build without a " +
				"real H.264 decoder (e.g. ffmpeg-free + noopenh264) cannot probe H.264 files.")
		}
		return nil, nil
	}
	return parseProbe(out)
}

// probeOutput is the part of `ffprobe -print_format json -show_format
// -show_streams` that is used.
type probeOutput struct {
	Format struct {
		Duration string            `json:"duration"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
		Width     *int   `json:"width"`
		Height    *int   `json:"height"`
	} `json:"streams"`
}

// parseProbe turns ffprobe output into the metadata of a video.
func parseProbe(out []byte) (*File, error) {
	var probe probeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}
	info := &File{Type: "video"}
	if seconds, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil && seconds > 0 {
		rounded := math.Round(seconds*1000) / 1000
		info.DurationSeconds = &rounded
	}
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			info.Resolution = &Resolution{Width: stream.Width, Height: stream.Height}
			info.Codec = stream.CodecName
			break
		}
	}
	tags := make(map[string]string, len(probe.Format.Tags))
	for key, value := range probe.Format.Tags {
		tags[strings.ToLower(key)] = value
	}
	first := func(names ...string) string {
		for _, name := range names {
			if tags[name] != "" {
				return tags[name]
			}
		}
		return ""
	}
	device := Device{
		Make:     first("com.apple.quicktime.make", "make"),
		Model:    first("com.apple.quicktime.model", "model"),
		Software: first("com.apple.quicktime.software", "encoder"),
	}
	if device != (Device{}) {
		info.Device = &device
	}
	info.TakenAt = tags["creation_time"]
	return info, nil
}
