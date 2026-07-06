// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	updaterplugin "github.com/SemRels/updater-composer/internal/plugin"
	"github.com/stretchr/testify/require"
)

func TestRunUpdatesComposerJSON(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	require.NoError(t, os.WriteFile(file, []byte("{\n  \"name\": \"demo/package\",\n  \"version\": \"1.0.0\"\n}\n"), 0o644))

	env := map[string]string{
		"SEMREL_VERSION":                     "v1.1.0",
		"SEMREL_PLUGIN_FILE":                 file,
		"SEMREL_PLUGIN_UPDATE_VERSION_FIELD": "true",
	}
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterplugin.NewUpdater()); code != 0 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	if !strings.Contains(string(got), `"version": "1.1.0"`) {
		t.Fatalf("updated file = %s", got)
	}
}

func TestRunNoOpByDefault(t *testing.T) {
	t.Parallel()

	env := map[string]string{"SEMREL_VERSION": "1.1.0"}
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterplugin.NewUpdater()); code != 0 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "disabled by default") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "left composer.json unchanged") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDryRunDoesNotWriteFile(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	original := "{\n  \"name\": \"demo/package\",\n  \"version\": \"1.0.0\"\n}\n"
	require.NoError(t, os.WriteFile(file, []byte(original), 0o644))

	env := map[string]string{
		"SEMREL_VERSION":                     "1.1.0",
		"SEMREL_DRY_RUN":                     "true",
		"SEMREL_PLUGIN_FILE":                 file,
		"SEMREL_PLUGIN_UPDATE_VERSION_FIELD": "true",
	}
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterplugin.NewUpdater()); code != 0 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[dry-run]") {
		t.Fatalf("stdout = %q", stdout.String())
	}

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	if string(got) != original {
		t.Fatalf("dry run modified file: %s", got)
	}
}

func TestRunMissingFileHandling(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"SEMREL_VERSION":                     "1.1.0",
		"SEMREL_PLUGIN_FILE":                 filepath.Join(testDir(t), "composer.json"),
		"SEMREL_PLUGIN_UPDATE_VERSION_FIELD": "true",
	}
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterplugin.NewUpdater()); code != 1 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "read") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunMalformedJSONHandling(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "composer.json")
	require.NoError(t, os.WriteFile(file, []byte("{"), 0o644))

	env := map[string]string{
		"SEMREL_VERSION":                     "1.1.0",
		"SEMREL_PLUGIN_FILE":                 file,
		"SEMREL_PLUGIN_UPDATE_VERSION_FIELD": "true",
	}
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterplugin.NewUpdater()); code != 1 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "parse") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunPathOverrideAlias(t *testing.T) {
	t.Parallel()

	dir := testDir(t)
	file := filepath.Join(dir, "nested-composer.json")
	require.NoError(t, os.WriteFile(file, []byte("{\n  \"name\": \"demo/package\",\n  \"version\": \"1.0.0\"\n}\n"), 0o644))

	env := map[string]string{
		"SEMREL_VERSION":                     "1.1.0",
		"SEMREL_PLUGIN_COMPOSER_FILE":        file,
		"SEMREL_PLUGIN_UPDATE_VERSION_FIELD": "true",
	}
	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(key string) string { return env[key] }, updaterplugin.NewUpdater()); code != 0 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}

	got, err := os.ReadFile(file)
	require.NoError(t, err)
	if !strings.Contains(string(got), `"version": "1.1.0"`) {
		t.Fatalf("updated file = %s", got)
	}
}

func TestRunRequiresVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if code := run(&stdout, &stderr, func(string) string { return "" }, updaterplugin.NewUpdater()); code != 1 {
		t.Fatalf("run() code = %d", code)
	}
	if !strings.Contains(stderr.String(), "SEMREL_VERSION is required") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func testDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp(".", ".updater-composer-main-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}
