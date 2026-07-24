// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The semrel Authors

package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	updaterplugin "github.com/SemRels/updater-composer/internal/plugin"
)

const pluginSchemaVersion = 1

type versionUpdater interface {
	Preview(path, version string, addIfMissing bool) (updaterplugin.Result, error)
	Update(path, version string, addIfMissing bool) (updaterplugin.Result, error)
}

func main() {
	os.Exit(run(os.Stdout, os.Stderr, os.Getenv, updaterplugin.NewUpdater()))
}

func run(stdout, stderr io.Writer, getenv func(string) string, updater versionUpdater) int {
	_, _ = fmt.Fprintf(stderr, "plugin_schema_version=%d\n", pluginSchemaVersion)

	version := getenv("SEMREL_VERSION")
	if version == "" {
		version = getenv("SEMREL_NEXT_VERSION")
	}
	if version == "" {
		_, _ = fmt.Fprintln(stderr, "updater-composer: SEMREL_VERSION is required")
		return 1
	}
	version = strings.TrimPrefix(version, "v")

	file := getenv("SEMREL_PLUGIN_FILE")
	if file == "" {
		file = getenv("SEMREL_PLUGIN_COMPOSER_FILE")
	}
	if file == "" {
		file = "composer.json"
	}

	if getenv("SEMREL_PLUGIN_UPDATE_VERSION_FIELD") != "true" {
		_, _ = fmt.Fprintln(stderr, "updater-composer: version field updates are disabled by default because Composer discourages hardcoded versions in composer.json; set SEMREL_PLUGIN_UPDATE_VERSION_FIELD=true to opt in")
		if getenv("SEMREL_DRY_RUN") == "true" {
			_, _ = fmt.Fprintf(stdout, "updater-composer: [dry-run] would leave %s unchanged\n", file)
		} else {
			_, _ = fmt.Fprintf(stdout, "updater-composer: left %s unchanged\n", file)
		}
		return 0
	}

	if getenv("SEMREL_DRY_RUN") == "true" {
		result, err := updater.Preview(file, version, true)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "updater-composer: %v\n", err)
			return 1
		}
		switch {
		case !result.Changed:
			_, _ = fmt.Fprintf(stdout, "updater-composer: [dry-run] %s already has version %s\n", file, version)
		case result.Added:
			_, _ = fmt.Fprintf(stdout, "updater-composer: [dry-run] would add version %s to %s\n", version, file)
		default:
			_, _ = fmt.Fprintf(stdout, "updater-composer: [dry-run] would update %s to version %s\n", file, version)
		}
		return 0
	}

	result, err := updater.Update(file, version, true)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "updater-composer: %v\n", err)
		return 1
	}

	switch {
	case !result.Changed:
		_, _ = fmt.Fprintf(stdout, "updater-composer: %s already has version %s\n", file, version)
	case result.Added:
		_, _ = fmt.Fprintf(stdout, "updater-composer: added version %s to %s\n", version, file)
	default:
		_, _ = fmt.Fprintf(stdout, "updater-composer: updated %s to version %s\n", file, version)
	}
	return 0
}
