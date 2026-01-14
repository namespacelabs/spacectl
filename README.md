<p>
  <a href="https://namespace.so">
    <img src="https://storage.googleapis.com/namespacelabs-docs-assets/gh/banner.svg" height="100">
  </a>
</p>

<p>
  <b><i>Namespace</i> is a development-optimized compute platform. It improves the performance and observability of Docker builds, GitHub Actions, and more, without requiring workflow changes. Learn more at https://namespace.so.</b>
</p>

# 🧑🏻‍🚀 space

`space` is a CLI designed to be run on Namespace runners, enabling various integrations.

## Installation

### From Release

Download the latest release from the [releases page](https://github.com/namespacelabs/space/releases).

### From Source

```bash
go install github.com/namespacelabs/space@latest
```

## Usage

```bash
# Display help
space --help

# Show version
space version
```

**--log-level:**

A global flag to change the log level across all sub commands. Accepts `debug, info, warn, error`.

### `space version`

Print the version number of the Space CLI.

```bash
space version
space version --output json
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--output, -o` | Output format: `plain` or `json`. Defaults to `plain`. |

### `space cache modes`

List available cache modes and whether they are detected in the current environment.

```bash
space cache modes
space cache modes -o json
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--output, -o` | Output format: `plain` or `json`. Defaults to `plain`. |

### `space cache mount`

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
space cache mount --detect='*'

# Detect only apt caches
space cache mount --detect=apt

# Use explicit go cache mode
space cache mount --mode=go

# Mount a specific path
space cache mount --path=/some/path

# Combine detection with explicit paths
space cache mount --detect='*' --path=/custom/cache
```

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

## Security

Please report security issues privately as described in [SECURITY.md](./SECURITY.md).
