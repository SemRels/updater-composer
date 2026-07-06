# updater-composer

[![Latest Release](https://img.shields.io/github/v/release/SemRels/updater-composer?label=version&color=blue)](https://github.com/SemRels/updater-composer/releases/latest)
[![License](https://img.shields.io/github/license/SemRels/updater-composer)](https://github.com/SemRels/updater-composer/blob/main/LICENSE)
[![CI](https://github.com/SemRels/updater-composer/actions/workflows/ci.yml/badge.svg)](https://github.com/SemRels/updater-composer/actions/workflows/ci.yml)
[![Scorecard](https://api.scorecard.dev/projects/github.com/SemRels/updater-composer/badge)](https://scorecard.dev/viewer/?uri=github.com/SemRels/updater-composer)

Updates the `version` field in `composer.json` when explicitly opted in.

Composer 2.x typically derives package versions from VCS tags, so hardcoding `version` in `composer.json` is discouraged. This plugin therefore follows the updater plugin subprocess contract used by `updater-npm`, but leaves the manifest unchanged unless `SEMREL_PLUGIN_UPDATE_VERSION_FIELD=true` is set.

This plugin is distributed as the standalone Go binary `semrel-plugin-updater-composer`. Semrel executes the binary as a subprocess, provides plugin configuration through `SEMREL_PLUGIN_*` environment variables, provides release context through `SEMREL_*` environment variables, reads standard output, and treats exit code `0` as success and any non-zero exit code as failure. Install the binary in `~/.semrel/plugins/` or anywhere on your `$PATH`.

## Installation

### Binary

```bash
go install github.com/SemRels/updater-composer/cmd/plugin@latest
```

### Docker

Pre-built, multi-platform images (linux/amd64, linux/arm64) are published to the GitHub Container Registry on every release:

```bash
docker pull ghcr.io/semrels/updater-composer:latest
```

Images are signed with [cosign](https://github.com/sigstore/cosign) and include a full SBOM attestation. Verify the signature:

```bash
cosign verify ghcr.io/semrels/updater-composer:latest \
  --certificate-identity-regexp 'https://github.com/SemRels/updater-composer/.github/workflows/release.yml.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Configuration

```yaml
plugins:
  - name: updater-composer
    path: ~/.semrel/plugins/semrel-plugin-updater-composer
    env:
      SEMREL_PLUGIN_FILE: "composer.json"
      SEMREL_PLUGIN_UPDATE_VERSION_FIELD: "true"
```

## `.semrel.yaml` example

```yaml
tagPrefix: "v"
version_ceiling: "1.0.0"
ceiling_strategy: clamp
commit_changelog: false
tag_exists_strategy: update-changelog

branches:
  - name: main

plugins:
  - uses: condition-github-actions
    phase: condition
  - name: updater-composer
    path: ~/.semrel/plugins/semrel-plugin-updater-composer
    env:
      SEMREL_PLUGIN_FILE: composer.json
      SEMREL_PLUGIN_UPDATE_VERSION_FIELD: "true"
```

## `SEMREL_PLUGIN_*` variables

| Name | Required | Description | Default |
| --- | --- | --- | --- |
| `SEMREL_PLUGIN_FILE` | Optional | Path to the Composer manifest to update. Mirrors `updater-npm`'s file-path contract. | composer.json |
| `SEMREL_PLUGIN_COMPOSER_FILE` | Optional | Alias for `SEMREL_PLUGIN_FILE` when a Composer-specific override name is preferred. | unset |
| `SEMREL_PLUGIN_UPDATE_VERSION_FIELD` | Optional | When `true`, updates or adds the `version` field in `composer.json`. Disabled by default because Composer prefers deriving versions from VCS tags. | false |

## `SEMREL_*` release context used

| Variable | Description |
| --- | --- |
| `SEMREL_VERSION` | Resolved release version for the current run. |
| `SEMREL_NEXT_VERSION` | Next version computed by semrel for the release. |
| `SEMREL_DRY_RUN` | Whether semrel is running in dry-run mode. |

## Example behavior

By default the plugin is a successful no-op and logs why it did not touch `composer.json`. When `SEMREL_PLUGIN_UPDATE_VERSION_FIELD=true`, it writes the next release version to the manifest, strips any leading `v` from `SEMREL_VERSION`, and preserves the existing key order and formatting as much as possible.

## License

Apache-2.0
