Messages package
================

This directory contains textual resources embedded into the `bundle` binary.
The `messages` package exposes helper functions to read these files at runtime.

Directory layout
----------------

- `use/` - very short "Usage" strings used for the cobra `Use` field. Example: `create`, `list`.
- `short/` - short one-line descriptions used in command summaries.
- `long/` - longer help text shown in command `--help` output.
- `templates/` - text templates used by various commands (suffixed with `.tmpl`).

Why this folder exists
----------------------

Embedding message files keeps the CLI self-contained and makes it easy to
localize or update help text without recompiling the code that consumes the
text. The messages are embedded using Go's `embed` package in
`messages/main.go`.

How to add or update messages
-----------------------------

1. Add or edit the appropriate file under `use/`, `short/` or `long/` using the
   command name as the filename. For example `use/create`, `short/create`,
   `long/create`.
2. If you add templates, place them under `templates/` and name them with the
   `.tmpl` suffix (for example `vm_confess_long.tmpl`).
3. Run `go test ./...` or `go build ./...` to ensure the embedded files are
   compiled into the binary. When running tests, the package will log an
   error if an embedded file is missing but will still return a default
   placeholder value (`undefined`).

Notes for developers
--------------------

- The package provides the following helpers:
  - `GetUse(name string) string`
  - `GetShort(name string) string`
  - `GetLong(name string) string`
  - `GetTemplate(name string) string`
- For command registration we read the `use` file for the command `Use` field.
  In tests or some environments where the message files may be missing,
  consider using a literal `Use` string in the command to avoid surprises.
