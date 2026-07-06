// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

// Package plugin updates composer.json files in-place.
package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

var indentationPattern = regexp.MustCompile(`(?m)^([ \t]+)"[^"]+"\s*:`)

// Result describes the requested change.
type Result struct {
	Changed bool
	Added   bool
}

// Updater updates composer.json version fields.
type Updater struct{}

// NewUpdater creates an updater.
func NewUpdater() *Updater {
	return &Updater{}
}

// Preview validates the composer.json file and computes the change without writing it.
func (u *Updater) Preview(path, version string, addIfMissing bool) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", path, err)
	}

	_, result, err := renderComposerJSON(data, version, addIfMissing)
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return result, nil
}

// Update rewrites the version field in composer.json.
func (u *Updater) Update(path, version string, addIfMissing bool) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read %s: %w", path, err)
	}

	updated, result, err := renderComposerJSON(data, version, addIfMissing)
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if !result.Changed {
		return result, nil
	}

	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", path, err)
	}
	return result, nil
}

func renderComposerJSON(data []byte, version string, addIfMissing bool) ([]byte, Result, error) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, Result{}, err
	}
	if _, ok := root.(map[string]any); !ok {
		return nil, Result{}, fmt.Errorf("top-level JSON value must be an object")
	}

	location, objectInfo, err := locateVersionField(data)
	if err != nil {
		return nil, Result{}, err
	}

	quoted := []byte(strconv.Quote(version))
	if location != nil {
		current := data[location.valueStart:location.valueEnd]
		if bytes.Equal(current, quoted) {
			return data, Result{Changed: false, Added: false}, nil
		}

		updated := make([]byte, 0, len(data)-len(current)+len(quoted))
		updated = append(updated, data[:location.valueStart]...)
		updated = append(updated, quoted...)
		updated = append(updated, data[location.valueEnd:]...)
		return updated, Result{Changed: true, Added: false}, nil
	}

	if !addIfMissing {
		return nil, Result{}, fmt.Errorf("version field not found in composer.json")
	}

	updated := insertVersionField(data, objectInfo, quoted)
	return updated, Result{Changed: true, Added: true}, nil
}

type fieldLocation struct {
	valueStart int
	valueEnd   int
}

type rootObjectInfo struct {
	start       int
	end         int
	empty       bool
	multiline   bool
	indentation string
	newline     string
}

func locateVersionField(data []byte) (*fieldLocation, rootObjectInfo, error) {
	i := skipWhitespace(data, 0)
	if i >= len(data) || data[i] != '{' {
		return nil, rootObjectInfo{}, fmt.Errorf("top-level JSON value must be an object")
	}

	info := rootObjectInfo{
		start:       i,
		multiline:   bytes.Contains(data, []byte("\n")),
		indentation: detectIndentation(data),
		newline:     detectNewline(data),
	}
	if info.newline == "" {
		info.newline = "\n"
	}

	i++
	for {
		i = skipWhitespace(data, i)
		if i >= len(data) {
			return nil, rootObjectInfo{}, fmt.Errorf("unexpected end of input")
		}
		if data[i] == '}' {
			info.end = i + 1
			info.empty = true
			return nil, info, nil
		}

		info.empty = false

		key, next, err := parseJSONString(data, i)
		if err != nil {
			return nil, rootObjectInfo{}, err
		}
		i = skipWhitespace(data, next)
		if i >= len(data) || data[i] != ':' {
			return nil, rootObjectInfo{}, fmt.Errorf("expected ':' after object key")
		}
		i = skipWhitespace(data, i+1)
		valueStart := i
		valueEnd, err := skipValue(data, i)
		if err != nil {
			return nil, rootObjectInfo{}, err
		}
		if key == "version" {
			info.end = findRootObjectEnd(data)
			return &fieldLocation{valueStart: valueStart, valueEnd: valueEnd}, info, nil
		}
		i = skipWhitespace(data, valueEnd)
		if i >= len(data) {
			return nil, rootObjectInfo{}, fmt.Errorf("unexpected end of input")
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			info.end = i + 1
			return nil, info, nil
		default:
			return nil, rootObjectInfo{}, fmt.Errorf("expected ',' or '}' in object")
		}
	}
}

func findRootObjectEnd(data []byte) int {
	i := skipWhitespace(data, 0)
	end, _ := skipObject(data, i)
	return end
}

func insertVersionField(data []byte, info rootObjectInfo, quotedVersion []byte) []byte {
	entry := append([]byte(`"version": `), quotedVersion...)
	if !info.multiline && info.indentation == "" && !bytes.Contains(data[info.start:info.end], []byte("\n")) {
		if info.empty {
			return append(append(data[:info.end-1], entry...), data[info.end-1:]...)
		}
		insertion := append([]byte(`, `), entry...)
		return append(append(data[:info.end-1], insertion...), data[info.end-1:]...)
	}

	indent := info.indentation
	if indent == "" {
		indent = "  "
	}

	braceIndex := info.end - 1
	whitespaceStart := braceIndex
	for whitespaceStart > info.start && isWhitespace(data[whitespaceStart-1]) {
		whitespaceStart--
	}
	closingWhitespace := data[whitespaceStart:braceIndex]
	if len(closingWhitespace) == 0 {
		closingWhitespace = []byte(info.newline)
	}

	updated := make([]byte, 0, len(data)+len(entry)+len(indent)+len(closingWhitespace)+2)
	updated = append(updated, data[:whitespaceStart]...)
	if !info.empty {
		updated = append(updated, ',')
	}
	updated = append(updated, closingWhitespace...)
	updated = append(updated, []byte(indent)...)
	updated = append(updated, entry...)
	updated = append(updated, closingWhitespace...)
	updated = append(updated, data[braceIndex:]...)
	return updated
}

func detectIndentation(data []byte) string {
	if match := indentationPattern.FindSubmatch(data); match != nil {
		return string(match[1])
	}
	return ""
}

func detectNewline(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	if bytes.Contains(data, []byte("\n")) {
		return "\n"
	}
	return ""
}

func skipWhitespace(data []byte, i int) int {
	for i < len(data) && isWhitespace(data[i]) {
		i++
	}
	return i
}

func isWhitespace(b byte) bool {
	return b == ' ' || b == '\n' || b == '\r' || b == '\t'
}

func parseJSONString(data []byte, start int) (string, int, error) {
	if start >= len(data) || data[start] != '"' {
		return "", 0, fmt.Errorf("expected string")
	}

	i := start + 1
	escaped := false
	for i < len(data) {
		switch data[i] {
		case '"':
			if !escaped {
				value, err := strconv.Unquote(string(data[start : i+1]))
				if err != nil {
					return "", 0, err
				}
				return value, i + 1, nil
			}
			escaped = false
		case '\\':
			escaped = !escaped
		default:
			escaped = false
		}
		i++
	}
	return "", 0, fmt.Errorf("unterminated string")
}

func skipValue(data []byte, start int) (int, error) {
	if start >= len(data) {
		return 0, fmt.Errorf("unexpected end of input")
	}

	switch data[start] {
	case '"':
		_, end, err := parseJSONString(data, start)
		return end, err
	case '{':
		return skipObject(data, start)
	case '[':
		return skipArray(data, start)
	case 't':
		return skipLiteral(data, start, "true")
	case 'f':
		return skipLiteral(data, start, "false")
	case 'n':
		return skipLiteral(data, start, "null")
	default:
		return skipNumber(data, start)
	}
}

func skipObject(data []byte, start int) (int, error) {
	i := start + 1
	for {
		i = skipWhitespace(data, i)
		if i >= len(data) {
			return 0, fmt.Errorf("unexpected end of input")
		}
		if data[i] == '}' {
			return i + 1, nil
		}
		_, next, err := parseJSONString(data, i)
		if err != nil {
			return 0, err
		}
		i = skipWhitespace(data, next)
		if i >= len(data) || data[i] != ':' {
			return 0, fmt.Errorf("expected ':' after object key")
		}
		i = skipWhitespace(data, i+1)
		next, err = skipValue(data, i)
		if err != nil {
			return 0, err
		}
		i = skipWhitespace(data, next)
		if i >= len(data) {
			return 0, fmt.Errorf("unexpected end of input")
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			return i + 1, nil
		default:
			return 0, fmt.Errorf("expected ',' or '}' in object")
		}
	}
}

func skipArray(data []byte, start int) (int, error) {
	i := start + 1
	for {
		i = skipWhitespace(data, i)
		if i >= len(data) {
			return 0, fmt.Errorf("unexpected end of input")
		}
		if data[i] == ']' {
			return i + 1, nil
		}
		next, err := skipValue(data, i)
		if err != nil {
			return 0, err
		}
		i = skipWhitespace(data, next)
		if i >= len(data) {
			return 0, fmt.Errorf("unexpected end of input")
		}
		switch data[i] {
		case ',':
			i++
		case ']':
			return i + 1, nil
		default:
			return 0, fmt.Errorf("expected ',' or ']' in array")
		}
	}
}

func skipLiteral(data []byte, start int, literal string) (int, error) {
	end := start + len(literal)
	if end > len(data) || string(data[start:end]) != literal {
		return 0, fmt.Errorf("invalid literal")
	}
	return end, nil
}

func skipNumber(data []byte, start int) (int, error) {
	i := start
	for i < len(data) {
		switch data[i] {
		case ' ', '\n', '\r', '\t', ',', '}', ']':
			return i, nil
		default:
			i++
		}
	}
	return i, nil
}
