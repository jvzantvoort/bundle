# Bundle Library — Functional and Structural Definition

## 1. Concept Overview
> The Bundle Library is the foundational component that defines what a bundle *is* and how it behaves.  
> A **bundle** is the smallest atomic unit in the Digital Asset Management (DAM) system.  
> A bundle encapsulates one or more *targets* (files or directories) and the metadata required to manage and verify them.

Key characteristics:
- **Atomicity:** No internal structural awareness beyond being a collection of targets.
- **Immutability:** Any modification results in a new bundle.
- **Deterministic Naming:** The physical name of a bundle is derived from its content checksums.
- **Self-describing:** The `.bundle` subdirectory contains all metadata and operational files.

---

## 2. Bundle Structure

### 2.1 Root Layout
Each bundle is a directory structured as follows:

```
bundle/
├── file1.jpg
├── file2.mp4
├── ...
└── .bundle/
    ├── META.json
    ├── TAGS.txt
    ├── SHA256SUM.txt
    └── STATE.json
```

### 2.2 `.bundle/` Directory Contents

| File | Description |
|------|--------------|
| `META.json` | Stores bundle metadata such as title, creation time, owner, and checksum aggregation info. |
| `TAGS.txt` | Plaintext or JSON list of user-assigned tags for classification. |
| `SHA256SUM.txt` | List of SHA256 checksums of all files in the bundle (excluding `.bundle/` itself). |
| `STATE.json` | Internal state file used by the Bundle Library for synchronization, indexing, and integrity status. |

---

## 3. Bundle Naming & Identification

### 3.1 Physical Name
Derived deterministically from the checksums of all contained files.

**Computation logic (formalized):**
1. Generate a SHA256 checksum for every file in the bundle, excluding `.bundle/`.
2. Collect and sort the hashes lexicographically.
3. Concatenate them with newline delimiters.
4. Compute the SHA256 of this concatenated string — that is the **bundle checksum**.
5. The bundle’s **physical name** is derived from this final hash.

**Example pseudocode (Python-like):**
```python
def compute_bundle_checksum(bundle_path):
    checksums = []
    for path in find_files(bundle_path, exclude=[".bundle", "SHA256SUM.txt"]):
        checksum = sha256sum(path)
        checksums.append(checksum)
    checksums.sort()
    combined = "\n".join(checksums).encode("utf-8")
    return sha256(combined).hexdigest()
```

### 3.2 Title
A human-readable name stored in `.bundle/META.json`:
```json
{
  "title": "Iceland Vacation 2024",
  "created_at": "2025-10-30T13:05:00Z",
  "bundle_checksum": "f63f83f3e5...",
  "author": "John",
  "version": 1
}
```

### 3.3 Identifier Relationship
| Type | Example | Purpose |
|------|----------|----------|
| Title | “Iceland Vacation 2024” | Human reference |
| Physical Name | `f63f83f3e5a9b...` | Content-derived unique identity |
| Path | `/mnt/nas/bundles/f63f83f3e5a9b...` | Physical storage location |

---

## 4. Metadata Components

### 4.1 TAGS.txt
Simple, appendable format for tagging:
```
travel
iceland
vacation
2024
photos
```
- Used for classification and search indexing.
- Can be imported/exported to/from the backend.

### 4.2 SHA256SUM.txt
Follows `sha256sum` standard output format:
```
0f343b0931126a20f133d67c2b018a3b  ./IMG_001.jpg
f3bbbd66a63d4bf1747940578ec3d010  ./IMG_002.jpg
...
```
- Serves as the canonical checksum record.
- Enables both verification and bundle checksum recomputation.

### 4.3 STATE.json
Stores operational state, e.g.:
```json
{
  "verified": true,
  "last_checked": "2025-10-30T14:02:00Z",
  "replicas": ["nas:/mnt/vol1", "cloud:s3://archive/iceland-2024"],
  "size_bytes": 12439123456
}
```

---

## 5. Library Functional Components

| Component | Description |
|------------|-------------|
| **BundleScanner** | Walks a directory tree to register all targets (excluding `.bundle`). |
| **ChecksumManager** | Computes file-level and bundle-level checksums; verifies integrity. |
| **MetadataManager** | Reads/writes `META.json`, `TAGS.txt`, `STATE.json`. |
| **TagManager** | Adds, removes, lists, and filters tags. |
| **IntegrityChecker** | Validates `SHA256SUM.txt` against actual files and bundle checksum. |
| **BundleSerializer** | Handles import/export of bundle metadata to/from backend APIs. |
| **StorageHandler** | Provides copy/sync/replicate methods to supported backends (local, NAS, cloud). |
| **LockManager** | Ensures thread-safe operations on `.bundle` directory (prevent concurrent mutation). |

---

## 6. Command Interfaces (Library Entry Points)
Intended for CLI and Backend use:

- `bundle create <path> --title "My Album"` → Creates `.bundle/`, computes checksums, generates metadata.
- `bundle verify <path>` → Recomputes and validates all checksums.
- `bundle info <path>` → Displays metadata, title, checksum, tag list, storage state.
- `bundle tag add <path> tagname` → Adds a tag to `.bundle/TAGS.txt`.
- `bundle rename <path> <new_title>` → Updates `META.json` title (does not change bundle checksum).

---

## 7. Principles of Operation
- **Immutable Data Model:** Changing *any* target file → new bundle with new checksum and physical name.
- **Self-contained Verification:** Everything needed to verify bundle integrity resides within the bundle itself.
- **Portable & Distributed:** Bundles can be copied or moved without losing metadata or breaking structure.
- **Content-Addressable:** Bundle identity depends solely on content — not on storage path, timestamps, or owner.

## 8. Preferences

My preferred language is golang. My code structure often resembles this:

* ``cmd/<command>/main.go``    # commands and sub commands (using cobra)
* ``config/main.go``           # configuration file handling (using viper)
* ``<function>/main.go``       # main block of functionality
* ``messages/long/<command>``  # for help messages using go:embed
* ``messages/usage/<command>`` # for usage messages using go:embed
* ``messages/short/<command>`` # for short messages using go:embed
* ``utils/*.go``               # for code that is shared between functional blocks

Preferred libraries:

* https://github.com/sirupsen/logrus for logging
* https://github.com/spf13/viper for configuration handling
* https://github.com/spf13/cobra for command line handling
* https://github.com/olekukonko/tablewriter for table writing
* https://github.com/fatih/color for colorizing text where needed


