---
name: ns-check-cobra
description: Checks `spf13/cobra` & `spf13/pflag` usage to make sure they are used consistently to Namespace standards.
---

# ns-check-cobra

This skill validates that Cobra CLI commands follow Namespace's conventions for flag definitions.

## Usage

```
ns-check-cobra [--fix]
```

Run this skill against command files to check for violations.

### Arguments

| Argument | Description |
|----------|-------------|
| `--fix` | Automatically fix all violations in-place. Updates flag names, output conventions, and related code references. |

### Behavior

**Without `--fix`:** Reports all violations without modifying files.

**With `--fix`:** Aggressively fixes all violations:
1. Renames flags to lowercase snake_case
2. Updates all references to renamed flags throughout the command file
3. Replaces `--json` flags with `--output`/`-o` pattern
4. Updates conditional checks (e.g., `if *jsonFlag` → `if *outputFlag == "json"`)
5. Sets output flag defaults to `"plain"` if incorrect

## Checks

### 1. Flag Naming (pflag)

Scan all flag definitions in the target files. Flag definitions use these pflag methods:

- `Flags().String()`, `Flags().StringP()`, `Flags().StringVar()`, `Flags().StringVarP()`
- `Flags().Bool()`, `Flags().BoolP()`, `Flags().BoolVar()`, `Flags().BoolVarP()`
- `Flags().Int()`, `Flags().IntP()`, `Flags().IntVar()`, `Flags().IntVarP()`
- `Flags().StringSlice()`, `Flags().StringSliceP()`, `Flags().StringSliceVar()`, `Flags().StringSliceVarP()`
- And similar for other types (`Float`, `Duration`, etc.)

**Rules:**

| Rule | Valid | Invalid |
|------|-------|---------|
| Flag names must be lowercase | `--cache_root` | `--Cache_Root`, `--CACHE_ROOT` |
| Flag names must use snake_case | `--dry_run`, `--cache_root` | `--dry-run`, `--dryRun`, `--DryRun` |

### 2. Output Flag Convention

When a command supports formatted output:

| Rule | Valid | Invalid |
|------|-------|---------|
| Must use `--output` with `-o` shorthand | `StringP("output", "o", ...)` | `String("format", ...)`, `StringP("out", "o", ...)` |
| Default value must be `"plain"` | `"plain"` | `"text"`, `"default"`, `""` |
| Must support `json` as an option | Check for `"json"` handling | Missing JSON support |
| Must NOT use `--json` flag | Use `--output json` | `Bool("json", ...)` |

## Output Format

Report violations in this format:

```
## Flag Naming Violations

- `file.go:42` - Flag `dry-run` should use snake_case: `dry_run`
- `file.go:58` - Flag `CacheRoot` must be lowercase snake_case: `cache_root`

## Output Flag Violations

- `file.go:30` - Uses `--format` instead of `--output`/`-o`
- `file.go:45` - Output flag default is `"text"`, should be `"plain"`
- `file.go:22` - Uses `--json` flag; prefer `--output json`

## Summary

Files checked: 3
Violations found: 5
```

If no violations are found:

```
## Summary

Files checked: 3
All checks passed ✓
```

## Fix Examples

### Snake Case Conversion

Before:
```go
dryRun := cmd.Flags().Bool("dry-run", false, "Skip actual changes")
```

After:
```go
dryRun := cmd.Flags().Bool("dry_run", false, "Skip actual changes")
```

### JSON Flag to Output Flag

Before:
```go
jsonFlag := cmd.Flags().BoolP("json", "j", false, "Output as JSON")

cmd.RunE = func(cmd *cobra.Command, args []string) error {
    if *jsonFlag {
        return outputJSON(w, result)
    }
    outputText(w, result)
    return nil
}
```

After:
```go
outputFlag := cmd.Flags().StringP("output", "o", "plain", "Output format: plain or json.")

cmd.RunE = func(cmd *cobra.Command, args []string) error {
    if *outputFlag == "json" {
        return outputJSON(w, result)
    }
    outputText(w, result)
    return nil
}
```
