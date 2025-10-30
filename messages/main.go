// Package messages provides embedded textual resources used by the CLI for
// help text, short/long descriptions and templates.
//
// The package embeds the files under the `messages/` directory and exposes
// small helper functions to read those resources by name. Embedding keeps the
// binary self-contained and makes it easy to localise or modify command
// text without touching Go source.
package messages

import (
	"embed"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

//go:embed long/* use/* short/* templates/*
var Content embed.FS

// GetContent returns the content of the embedded message file in `folder`.
// When the requested file is missing the function logs an error and returns
// the string "undefined". The returned string has any trailing newline
// removed.
func GetContent(folder, name string) string {
	filename := fmt.Sprintf("%s/%s", folder, name)

	msgstr, err := Content.ReadFile(filename)
	if err != nil {
		// We log the error to aid debugging. Tests and the CLI are resilient to
		// missing message files, but it's helpful to record the cause.
		log.Errorf("%s", err)
		msgstr = []byte("undefined")
	}
	return strings.TrimSuffix(string(msgstr), "\n")
}

// GetShort returns the short description for a command (from messages/short).
func GetShort(name string) string {
	return GetContent("short", name)
}

// GetUse returns the 'use' string for a command (from messages/use).
// This is typically the single-word command name used by cobra.
func GetUse(name string) string {
	return GetContent("use", name)
}

// GetLong returns the long description for a command (from messages/long).
func GetLong(name string) string {
	return GetContent("long", name)
}

// GetTemplate returns a named template content under messages/templates.
// The name passed is combined with the ".tmpl" suffix.
func GetTemplate(name string) string {
	return GetContent("templates", name+".tmpl")
}
