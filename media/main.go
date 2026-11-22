// Package media keeps the media metadata of a bundle's files: EXIF data,
// resolution, capture time, GPS position and the device (camera or phone)
// that made them.
//
// The metadata is stored in .bundle/MEDIA.json, separate from META.json, in
// the format bundle-python writes, so either tool can refresh a file the
// other one wrote. .bundle/ is outside the bundle checksum: scanning never
// changes a bundle's identity.
//
// Images are read with the Go image decoders (JPEG, PNG, GIF, BMP, TIFF and
// WebP) and EXIF data with goexif (JPEG and TIFF). Videos need ffprobe from
// ffmpeg on PATH; without it they are skipped with a warning.
//
// Example usage:
//
//	b, err := bundle.Load("/path/to/bundle")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	summary, err := media.Scan(b.Path, b.Files.Records, media.Options{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("%d added, %d unchanged\n", summary.Added, summary.Skipped)
package media

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jvzantvoort/bundle/checksum"
	"github.com/jvzantvoort/bundle/lock"
	log "github.com/sirupsen/logrus"
)

// FileName is the name of the media metadata file in .bundle/.
const FileName = "MEDIA.json"

// SchemaVersion is the version of the MEDIA.json format this package writes.
const SchemaVersion = 1

// Resolution is the displayed size of an image or video, with the EXIF
// orientation applied. ffprobe can leave either dimension unknown.
type Resolution struct {
	Width  *int `json:"width"`
	Height *int `json:"height"`
}

// GPS is a position in decimal degrees, rounded to six decimals.
type GPS struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Device is the camera, phone or other device recorded in a file. Files is
// only set in Document.Devices, where it counts the files made with it.
type Device struct {
	Make     string `json:"make,omitempty"`
	Model    string `json:"model,omitempty"`
	Lens     string `json:"lens,omitempty"`
	Software string `json:"software,omitempty"`
	Files    int    `json:"files,omitempty"`
}

// File is the media metadata of one file.
type File struct {
	Type            string         `json:"type"`             // "image" or "video"
	Format          string         `json:"format,omitempty"` // image format, e.g. "JPEG"
	Resolution      *Resolution    `json:"resolution,omitempty"`
	TakenAt         string         `json:"taken_at,omitempty"` // as recorded: EXIF "2006:01:02 15:04:05" or ISO 8601
	Device          *Device        `json:"device,omitempty"`
	GPS             *GPS           `json:"gps,omitempty"`
	DurationSeconds *float64       `json:"duration_seconds,omitempty"` // videos only
	Codec           string         `json:"codec,omitempty"`            // videos only
	SHA256          string         `json:"sha256,omitempty"`           // checksum of the file when it was scanned
	EXIF            map[string]any `json:"exif,omitempty"`
}

// Document is the content of MEDIA.json. Files maps a path relative to the
// bundle root to its metadata.
type Document struct {
	Version   int             `json:"version"`
	Generator string          `json:"generator,omitempty"`
	UpdatedAt string          `json:"updated_at,omitempty"`
	Devices   []Device        `json:"devices"`
	Files     map[string]File `json:"files"`
}

// Options control Scan.
type Options struct {
	// Force extracts every file again, also when its checksum is unchanged.
	Force bool
	// Generator names the program in MEDIA.json, e.g. "bundle 1.2.0".
	Generator string
}

// Summary reports what Scan changed.
type Summary struct {
	Added       int `json:"added"`
	Updated     int `json:"updated"`
	Skipped     int `json:"skipped"`     // checksum unchanged since the last scan
	Unsupported int `json:"unsupported"` // not an image or video this package can read
	Removed     int `json:"removed"`     // entries of files no longer in the bundle
	Files       int `json:"files"`       // media files in MEDIA.json after the scan
	Devices     int `json:"devices"`
}

// Path returns the location of MEDIA.json in a bundle.
func Path(bundlePath string) string {
	return filepath.Join(bundlePath, ".bundle", FileName)
}

// Load reads MEDIA.json of a bundle. A bundle without one gives an empty
// document, not an error.
//
// Parameters:
//   - bundlePath: path to the bundle directory
//
// Returns:
//   - *Document: the stored metadata, or an empty document
//   - error: if MEDIA.json cannot be read or parsed
func Load(bundlePath string) (*Document, error) {
	doc := &Document{Version: SchemaVersion, Devices: []Device{}, Files: map[string]File{}}
	content, err := os.ReadFile(Path(bundlePath))
	if errors.Is(err, fs.ErrNotExist) {
		return doc, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(content, doc); err != nil {
		// Older bundle-python versions wrote NaN for an EXIF rational with a
		// zero denominator, which is not valid JSON: read those values as null
		fixed, changed := nonFiniteToNull(content)
		if !changed || json.Unmarshal(fixed, doc) != nil {
			return nil, fmt.Errorf("parse %s: %w", FileName, err)
		}
		log.Debugf("%s: read NaN or Infinity values as null", Path(bundlePath))
	}
	if doc.Files == nil {
		doc.Files = map[string]File{}
	}
	if doc.Devices == nil {
		doc.Devices = []Device{}
	}
	return doc, nil
}

// nonFiniteToNull replaces the NaN, Infinity and -Infinity tokens that
// Python's json module writes with null, outside of strings.
func nonFiniteToNull(content []byte) ([]byte, bool) {
	out := make([]byte, 0, len(content))
	inString, escaped, changed := false, false, false
	for i := 0; i < len(content); i++ {
		c := content[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			out = append(out, c)
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		replaced := false
		for _, token := range []string{"NaN", "Infinity", "-Infinity"} {
			if bytes.HasPrefix(content[i:], []byte(token)) {
				out = append(out, "null"...)
				i += len(token) - 1
				replaced, changed = true, true
				break
			}
		}
		if !replaced {
			out = append(out, c)
		}
	}
	return out, changed
}

// Save writes MEDIA.json atomically, with sorted keys and two-space
// indentation like bundle-python.
//
// Parameters:
//   - bundlePath: path to the bundle directory
//   - doc: the metadata to store
//
// Returns:
//   - error: if the file cannot be written
func Save(bundlePath string, doc *Document) error {
	content, err := marshalSorted(doc)
	if err != nil {
		return err
	}
	target := Path(bundlePath)
	tmp, err := os.CreateTemp(filepath.Dir(target), ".media-")
	if err != nil {
		return err
	}
	_, writeErr := tmp.Write(content)
	if err := errors.Join(writeErr, tmp.Close(), os.Chmod(tmp.Name(), 0o644)); err != nil {
		return errors.Join(err, os.Remove(tmp.Name()))
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return errors.Join(err, os.Remove(tmp.Name()))
	}
	return nil
}

// marshalSorted renders doc with every object's keys sorted, by passing it
// through generic maps (encoding/json sorts map keys).
func marshalSorted(doc *Document) ([]byte, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var generic any
	if err := decoder.Decode(&generic); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(generic); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Scan extracts the media metadata of the files in a bundle and stores it in
// MEDIA.json. It is incremental: a file whose checksum matches the stored
// one is skipped unless opts.Force is set, and entries of files that are no
// longer in the manifest are removed. The bundle is locked while scanning.
//
// Parameters:
//   - bundlePath: path to the bundle directory
//   - records: the bundle manifest (bundle.Files.Records)
//   - opts: scan options
//
// Returns:
//   - *Summary: counts of what changed
//   - error: lock contention, or a file or MEDIA.json that cannot be read or written
func Scan(bundlePath string, records []checksum.ChecksumRecord, opts Options) (*Summary, error) {
	bundleLock, err := lock.AcquireLock(bundlePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := bundleLock.Release(); err != nil {
			log.Errorf("failed to release lock: %v", err)
		}
	}()

	doc, err := Load(bundlePath)
	if err != nil {
		return nil, err
	}
	sorted := append([]checksum.ChecksumRecord(nil), records...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FilePath < sorted[j].FilePath })

	summary := &Summary{}
	ex := &extractor{}
	seen := make(map[string]bool, len(sorted))
	for _, record := range sorted {
		seen[record.FilePath] = true
		old, known := doc.Files[record.FilePath]
		if known && !opts.Force && old.SHA256 != "" && old.SHA256 == record.Checksum {
			summary.Skipped++
			continue
		}
		full := filepath.Join(bundlePath, filepath.FromSlash(record.FilePath))
		log.Debugf("extracting media metadata of %s", full)
		info, err := ex.extract(full)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", full, err)
		}
		if info == nil {
			// Keep an entry for a file that became unreadable for this tool
			// (bundle-python may support more formats): only stale entries go
			if known && old.SHA256 != record.Checksum {
				delete(doc.Files, record.FilePath)
				summary.Removed++
			}
			summary.Unsupported++
			continue
		}
		info.SHA256 = record.Checksum
		doc.Files[record.FilePath] = *info
		if known {
			summary.Updated++
		} else {
			summary.Added++
		}
	}
	for path := range doc.Files {
		if !seen[path] {
			delete(doc.Files, path)
			summary.Removed++
		}
	}

	doc.Version = SchemaVersion
	doc.Generator = opts.Generator
	doc.UpdatedAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	doc.Devices = Devices(doc)
	if err := Save(bundlePath, doc); err != nil {
		return nil, err
	}
	summary.Files = len(doc.Files)
	summary.Devices = len(doc.Devices)
	log.Infof("media metadata of %s: %d added, %d updated, %d unchanged, %d unsupported, %d removed",
		bundlePath, summary.Added, summary.Updated, summary.Skipped, summary.Unsupported, summary.Removed)
	return summary, nil
}

// Devices returns the distinct devices of a document with the number of
// files made with each, most used first.
func Devices(doc *Document) []Device {
	counts := map[Device]int{}
	for _, file := range doc.Files {
		if file.Device == nil {
			continue
		}
		key := *file.Device
		key.Files = 0
		counts[key]++
	}
	devices := make([]Device, 0, len(counts))
	for device, count := range counts {
		device.Files = count
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		a, b := devices[i], devices[j]
		if a.Files != b.Files {
			return a.Files > b.Files
		}
		if a.Make != b.Make {
			return a.Make < b.Make
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		return strings.Compare(a.Lens+"\x00"+a.Software, b.Lens+"\x00"+b.Software) < 0
	})
	return devices
}
