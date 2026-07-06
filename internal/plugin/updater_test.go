// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdaterUpdateExistingVersion(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	original := "{\n    \"name\": \"demo/package\",\n    \"version\": \"1.2.3\",\n    \"type\": \"project\"\n}\n"
	require.NoError(t, os.WriteFile(file, []byte(original), 0o644))

	result, err := NewUpdater().Update(file, "1.3.0", true)
	require.NoError(t, err)
	if !result.Changed || result.Added {
		t.Fatalf("unexpected result = %+v", result)
	}

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	if !strings.Contains(string(got), `    "version": "1.3.0",`) {
		t.Fatalf("updated file = %s", got)
	}
}

func TestUpdaterAddsMissingVersionWhenOptedIn(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	require.NoError(t, os.WriteFile(file, []byte("{\n  \"name\": \"demo/package\"\n}\n"), 0o644))

	result, err := NewUpdater().Update(file, "1.3.0", true)
	require.NoError(t, err)
	if !result.Changed || !result.Added {
		t.Fatalf("unexpected result = %+v", result)
	}

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	if !strings.Contains(string(got), "\n  \"version\": \"1.3.0\"\n") {
		t.Fatalf("updated file = %s", got)
	}
}

func TestUpdaterPreviewDoesNotWriteFile(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	original := "{\n  \"name\": \"demo/package\",\n  \"version\": \"1.0.0\"\n}\n"
	require.NoError(t, os.WriteFile(file, []byte(original), 0o644))

	result, err := NewUpdater().Preview(file, "1.1.0", true)
	require.NoError(t, err)
	if !result.Changed {
		t.Fatalf("expected change, got %+v", result)
	}

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	if string(got) != original {
		t.Fatalf("Preview() modified file: %s", got)
	}
}

func TestUpdaterMissingFile(t *testing.T) {
	t.Parallel()

	_, err := NewUpdater().Update(filepath.Join(testDir(t), "composer.json"), "1.3.0", true)
	if err == nil || !strings.Contains(err.Error(), "read") {
		t.Fatalf("expected read error, got %v", err)
	}
}

func TestUpdaterInvalidJSON(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	require.NoError(t, os.WriteFile(file, []byte("{"), 0o644))

	_, err := NewUpdater().Update(file, "1.3.0", true)
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func testDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp(".", ".updater-composer-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}
