<p>
  <a href="https://namespace.so">
    <img src="https://storage.googleapis.com/namespacelabs-docs-assets/gh/banner.svg" height="100">
  </a>
</p>

<p>
  <b><i>Namespace</i> is a development-optimized compute platform. It improves the performance and observability of Docker builds, GitHub Actions, and more, without requiring workflow changes. Learn more at https://namespace.so.</b>
</p>

# 🧑🏻‍🚀 spacectl

`spacectl` is a CLI designed to be run on Namespace runners, enabling various integrations.

## Installation

### From Release

Download the latest release from the [releases page](https://github.com/namespacelabs/spacectl/releases).

### From Source

```bash
go install github.com/namespacelabs/spacectl@latest
```

## Usage

```bash
# Display help
spacectl --help

# Show version
spacectl version
```

**--log-level:**

A global flag to change the log level across all sub commands. Accepts `debug, info, warn, error`.

### `spacectl version`

Print the version number of the spacectl CLI.

```bash
spacectl version
spacectl version --output json
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--output, -o` | Output format: `plain` or `json`. Defaults to `plain`. |

### `spacectl cache modes`

List available cache modes and whether they are detected in the current environment.

```bash
spacectl cache modes
spacectl cache modes -o json
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--output, -o` | Output format: `plain` or `json`. Defaults to `plain`. |

### `spacectl cache mount`

Restore cache paths from a Namespace volume.

**Flags:**

| Flag | Description |
|------|-------------|
| `--detect` | Detects cache mode(s) based on environment. Use `--detect='*'` to enable all detectors, or specify individual modes like `--detect=apt`. Can be specified multiple times. |
| `--mode` | Explicit cache mode(s) to enable (e.g., `--mode=go`). Can be specified multiple times. |
| `--path` | Explicit cache path(s) to enable (e.g., `--path=/some/path`). Can be specified multiple times. |
| `--cache_root` | Override the root path where cache volumes are mounted. Defaults to `$NSC_CACHE_PATH`. |
| `--dry_run` | If true, mounting is skipped and only reports what would be done. Defaults to `true` outside CI, `false` in CI (GitHub Actions, GitLab CI). |
| `--eval_file` | Write a file that can be sourced to export environment variables. |
| `--output, -o` | Output format: `plain` or `json`. Defaults to `plain`. |

**Examples:**

```bash
# Detect all available cache modes
spacectl cache mount --detect='*'

# Detect only apt caches
spacectl cache mount --detect=apt

# Use explicit go cache mode
spacectl cache mount --mode=go

# Mount a specific path
spacectl cache mount --path=/some/path

# Combine detection with explicit paths
spacectl cache mount --detect='*' --path=/custom/cache
```

### `spacectl cache post`

Run post-step cache cleanup, just before the cache volume is persisted for the
next run. Removes transient install state that modes mark for post-step removal
(currently the `lix` mode's `/nix/receipt.json`), so it is not carried across
runs (the `lix` mode marks `/nix/receipt.json` and `/etc/nix/nix.conf`).
Invoke it from a cache action's post step with the same mode selection used at
mount time.

**Flags:**

| Flag | Description |
|------|-------------|
| `--detect` | Detect cache mode(s) based on environment. Use `--detect='*'` to enable all detectors. Can be specified multiple times. |
| `--mode` | Explicit cache mode(s) to enable (e.g., `--mode=lix`). Can be specified multiple times. |
| `--cache_root` | Override the root path where cache volumes are mounted. Defaults to `$NSC_CACHE_PATH`. |
| `--dry_run` | If true, removal is skipped and only reports what would be done. Defaults to `true` outside CI, `false` in CI. |
| `--output, -o` | Output format: `plain` or `json`. Defaults to `plain`. |

```bash
# Mount step (job start)
spacectl cache mount --mode=lix

# Post step (job end) — drop the lix install state before persisting
spacectl cache post --mode=lix
```

See [PROPOSAL-lix.md](./PROPOSAL-lix.md) for the design rationale behind the
`lix` mode.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## Security

Please report security issues privately as described in [SECURITY.md](./SECURITY.md).
