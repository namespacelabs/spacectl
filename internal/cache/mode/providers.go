package mode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

// AptProvider

const (
	aptDirCacheKey         = "Dir::Cache"
	aptDirCacheArchivesKey = "Dir::Cache::archives"
	aptDirEtcKey           = "Dir::Etc"
	aptDirEtcPartsKey      = "Dir::Etc::parts"
)

var aptConfigRegex = regexp.MustCompile(`(.+)\s"(.*)";`)

type AptProvider struct{}

func (p AptProvider) Name() string {
	return "apt"
}

func (p AptProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("apt-config"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath apt-config: %w", err)
	}
	return true, nil
}

func (p AptProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "apt-config", "dump")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, err
	}

	aptConfig := make(map[string]string, 4)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		result := aptConfigRegex.FindStringSubmatch(line)
		if len(result) != 3 {
			continue
		}

		switch result[1] {
		case aptDirCacheKey, aptDirCacheArchivesKey, aptDirEtcKey, aptDirEtcPartsKey:
			aptConfig[result[1]] = result[2]
		default:
			continue
		}
	}
	if scanner.Err() != nil {
		return PlanResult{}, fmt.Errorf("scanning apt-config output: %w", scanner.Err())
	}

	if _, ok := aptConfig[aptDirCacheKey]; !ok {
		return PlanResult{}, fmt.Errorf(aptDirCacheKey + " not found in apt-config output")
	}
	if _, ok := aptConfig[aptDirCacheArchivesKey]; !ok {
		return PlanResult{}, fmt.Errorf(aptDirCacheArchivesKey + " not found in apt-config output")
	}

	result := PlanResult{
		MountPaths: []string{
			fmt.Sprintf("/%s/%s", aptConfig[aptDirCacheKey], aptConfig[aptDirCacheArchivesKey]),
		},
	}

	// remove docker-clean script
	if aptConfig[aptDirEtcKey] != "" && aptConfig[aptDirEtcPartsKey] != "" {
		path := fmt.Sprintf("/%s/%s/docker-clean", aptConfig[aptDirEtcKey], aptConfig[aptDirEtcPartsKey])
		_, err := req.Exec.Stat(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return PlanResult{}, fmt.Errorf("stat docker-clean script: %w", err)
		}
		if err == nil {
			result.RemovePaths = append(result.RemovePaths, path)
		}
	}

	return result, nil
}

// BrewProvider

const brewfile = "Brewfile"

type BrewProvider struct{}

func (p BrewProvider) Name() string {
	return "brew"
}

func (p BrewProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("brew"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath brew: %w", err)
	}

	if _, err := req.Exec.Stat(brewfile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", brewfile, err)
	}

	return true, nil
}

func (p BrewProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "brew", "--cache")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("brew --cache: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from brew --cache")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}

// BunProvider

const bunLockFile = "bun.lock"

type BunProvider struct{}

func (p BunProvider) Name() string {
	return "bun"
}

func (p BunProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("bun"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath bun: %w", err)
	}

	if _, err := req.Exec.Stat(bunLockFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", bunLockFile, err)
	}

	return true, nil
}

func (p BunProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "bun", "pm", "cache")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("bun pm cache: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from bun pm cache")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}

// CocoapodsProvider

const (
	cocoapodsCachePath = "~/Library/Caches/CocoaPods"
	cocoapodsPodfile   = "Podfile"
)

type CocoapodsProvider struct{}

func (p CocoapodsProvider) Name() string {
	return "cocoapods"
}

func (p CocoapodsProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("pod"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath pod: %w", err)
	}

	if _, err := req.Exec.Stat(cocoapodsPodfile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", cocoapodsPodfile, err)
	}

	return true, nil
}

func (p CocoapodsProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	return PlanResult{
		MountPaths: []string{
			"./Pods",
			cocoapodsCachePath,
		},
	}, nil
}

// ComposerProvider

const composerJsonFile = "composer.json"

type ComposerProvider struct{}

func (p ComposerProvider) Name() string {
	return "composer"
}

func (p ComposerProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("composer"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath composer: %w", err)
	}

	if _, err := req.Exec.Stat(composerJsonFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", composerJsonFile, err)
	}

	return true, nil
}

func (p ComposerProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "composer", "config", "--global", "cache-files-dir")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("composer config --global cache-files-dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from composer config")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}

// DenoProvider

const (
	denoDirKey   = "denoDir"
	denoLockFile = "deno.lock"
)

type DenoProvider struct{}

func (p DenoProvider) Name() string {
	return "deno"
}

func (p DenoProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("deno"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath deno: %w", err)
	}

	if _, err := req.Exec.Stat(denoLockFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", denoLockFile, err)
	}

	return true, nil
}

func (p DenoProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "deno", "info", "--json")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("deno info --json: %w", err)
	}

	var denoInfo map[string]any
	if err := json.Unmarshal(output, &denoInfo); err != nil {
		return PlanResult{}, fmt.Errorf("parse deno info output: %w", err)
	}

	denoDir, ok := denoInfo[denoDirKey].(string)
	if !ok || denoDir == "" {
		return PlanResult{}, fmt.Errorf("denoDir not found in deno info output")
	}

	return PlanResult{
		MountPaths: []string{denoDir},
	}, nil
}

// GoProvider

const (
	goCacheKey     = "GOCACHE"
	goModeCacheKey = "GOMODCACHE"
	goModFile      = "go.mod"
	goWorkFile     = "go.work"
)

type GoProvider struct{}

func (p GoProvider) Name() string {
	return "go"
}

func (p GoProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("go"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath go: %w", err)
	}

	if _, err := req.Exec.Stat(goModFile); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", goModFile, err)
	}

	if _, err := req.Exec.Stat(goWorkFile); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", goWorkFile, err)
	}

	return false, nil
}

func (p GoProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "-json", goCacheKey, goModeCacheKey)
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("go env: %w", err)
	}

	var goEnv map[string]string
	if err := json.Unmarshal(output, &goEnv); err != nil {
		return PlanResult{}, fmt.Errorf("parse go env output: %w", err)
	}

	if _, ok := goEnv[goCacheKey]; !ok {
		return PlanResult{}, fmt.Errorf(goCacheKey + " not found in go env output")
	}
	if _, ok := goEnv[goModeCacheKey]; !ok {
		return PlanResult{}, fmt.Errorf(goModeCacheKey + " not found in go env output")
	}

	return PlanResult{
		MountPaths: []string{goEnv[goCacheKey], goEnv[goModeCacheKey]},
	}, nil
}

// GolangCILintProvider

const (
	golangCILintCacheDirPrefix  = "dir:"
	golangCILintDefaultCacheDir = "~/.cache/golangci-lint"
	golangCILintConfigYml       = ".golangci.yml"
	golangCILintConfigYaml      = ".golangci.yaml"
)

type GolangCILintProvider struct{}

func (p GolangCILintProvider) Name() string {
	return "golangci-lint"
}

func (p GolangCILintProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("golangci-lint"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath golangci-lint: %w", err)
	}

	if _, err := req.Exec.Stat(golangCILintConfigYml); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", golangCILintConfigYml, err)
	}

	if _, err := req.Exec.Stat(golangCILintConfigYaml); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", golangCILintConfigYaml, err)
	}

	return false, nil
}

func (p GolangCILintProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "golangci-lint", "cache", "status")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("golangci-lint cache status: %w", err)
	}

	cacheDir := golangCILintDefaultCacheDir
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(strings.ToLower(line), golangCILintCacheDirPrefix) {
			cacheDir = strings.TrimSpace(line[len(golangCILintCacheDirPrefix):])
			break
		}
	}
	if scanner.Err() != nil {
		return PlanResult{}, fmt.Errorf("scanning golangci-lint output: %w", scanner.Err())
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}

// GradleProvider

const (
	gradleCachesPath  = "~/.gradle/caches"
	gradleWrapperPath = "~/.gradle/wrapper"
	gradlewFile       = "gradlew"
	buildGradleFile   = "build.gradle"
)

type GradleProvider struct{}

func (p GradleProvider) Name() string {
	return "gradle"
}

func (p GradleProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("gradle"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath gradle: %w", err)
	}

	if _, err := req.Exec.Stat(gradlewFile); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", gradlewFile, err)
	}

	if _, err := req.Exec.Stat(buildGradleFile); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", buildGradleFile, err)
	}

	return false, nil
}

func (p GradleProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	return PlanResult{
		MountPaths: []string{
			gradleCachesPath,
			gradleWrapperPath,
		},
	}, nil
}

// MavenProvider

const (
	mavenRepositoryPath = "~/.m2/repository"
	mavenPomFile        = "pom.xml"
)

type MavenProvider struct{}

func (p MavenProvider) Name() string {
	return "maven"
}

func (p MavenProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("mvn"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath mvn: %w", err)
	}

	if _, err := req.Exec.Stat(mavenPomFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", mavenPomFile, err)
	}

	return true, nil
}

func (p MavenProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	return PlanResult{
		MountPaths: []string{mavenRepositoryPath},
	}, nil
}

// MiseProvider

const (
	miseDataDirKey      = "MISE_DATA_DIR"
	miseXdgDataHomeKey  = "XDG_DATA_HOME"
	miseLocalAppDataKey = "LOCALAPPDATA"
	miseDefaultPath     = "mise"
)

var miseConfigFiles = []string{
	"mise.toml",
	".mise.toml",
	".tool-versions",
	"mise/config.toml",
	".mise/config.toml",
	".config/mise.toml",
	".config/mise/config.toml",
}

type MiseProvider struct{}

func (p MiseProvider) Name() string {
	return "mise"
}

func (p MiseProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("mise"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath mise: %w", err)
	}

	for _, configFile := range miseConfigFiles {
		if _, err := req.Exec.Stat(configFile); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("stat %s: %w", configFile, err)
		}
	}

	return false, nil
}

func (p MiseProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	var mountTarget string
	if dir := os.Getenv(miseDataDirKey); dir != "" {
		mountTarget = dir
	} else if xdgDataHome := os.Getenv(miseXdgDataHomeKey); xdgDataHome != "" {
		mountTarget = filepath.Join(xdgDataHome, miseDefaultPath)
	} else if runtime.GOOS == "windows" {
		if localAppData := os.Getenv(miseLocalAppDataKey); localAppData != "" {
			mountTarget = filepath.Join(localAppData, miseDefaultPath)
		}
	}

	if mountTarget == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return PlanResult{}, fmt.Errorf("get user home dir: %w", err)
		}
		mountTarget = filepath.Join(homeDir, ".local", "share", miseDefaultPath)
	}

	return PlanResult{
		MountPaths: []string{mountTarget},
	}, nil
}

// NixProvider

const nixCachePath = "~/.cache/nix"

var nixProjectFiles = []string{
	"flake.nix",
	"shell.nix",
	"default.nix",
}

type NixProvider struct{}

func (p NixProvider) Name() string {
	return "nix"
}

func (p NixProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("nix"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath nix: %w", err)
	}

	for _, projectFile := range nixProjectFiles {
		if _, err := req.Exec.Stat(projectFile); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("stat %s: %w", projectFile, err)
		}
	}

	return false, nil
}

func (p NixProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	return PlanResult{
		MountPaths: []string{
			nixCachePath,
			"/nix",
		},
	}, nil
}

// PlaywrightProvider

const (
	playwrightBrowsersPathKey  = "PLAYWRIGHT_BROWSERS_PATH"
	playwrightDefaultCachePath = "~/.cache/ms-playwright"
	playwrightDarwinCachePath  = "~/Library/Caches/ms-playwright"
	playwrightWindowsCachePath = "%USERPROFILE%\\AppData\\Local\\ms-playwright"
)

type PlaywrightProvider struct{}

func (p PlaywrightProvider) Name() string {
	return "playwright"
}

func (p PlaywrightProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("playwright"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath playwright: %w", err)
	}
	return true, nil
}

func (p PlaywrightProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	if browsersPath := os.Getenv(playwrightBrowsersPathKey); browsersPath != "" {
		return PlanResult{
			MountPaths: []string{browsersPath},
		}, nil
	}

	var mountTarget string
	switch runtime.GOOS {
	case "darwin":
		mountTarget = playwrightDarwinCachePath
	case "windows":
		mountTarget = playwrightWindowsCachePath
	default:
		mountTarget = playwrightDefaultCachePath
	}

	return PlanResult{
		MountPaths: []string{mountTarget},
	}, nil
}

// PnpmProvider

const (
	pnpmPackageImportMethodKey   = "npm_config_package_import_method"
	pnpmPackageImportMethodValue = "copy"
	pnpmWarningFixVersion        = "v9.7.0"
	pnpmWarningPrefix            = "\u2009WARN\u2009" // thin space + WARN + thin space
	pnpmLockFile                 = "pnpm-lock.yaml"
)

type PnpmProvider struct{}

func (p PnpmProvider) Name() string {
	return "pnpm"
}

func (p PnpmProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("pnpm"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath pnpm: %w", err)
	}

	if _, err := req.Exec.Stat(pnpmLockFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", pnpmLockFile, err)
	}

	return true, nil
}

func (p PnpmProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	versionCmd := exec.CommandContext(ctx, "pnpm", "--version")
	versionOutput, err := req.Exec.Output(versionCmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("pnpm --version: %w", err)
	}
	// pnpm < 9.7.0 prints warnings to stdout, so only the last line contains the version.
	versionLines := strings.Split(strings.TrimSpace(string(versionOutput)), "\n")
	version := "v" + strings.TrimSpace(versionLines[len(versionLines)-1])

	cmd := exec.CommandContext(ctx, "pnpm", "store", "path", "--loglevel", "error")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("pnpm store path: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if semver.Compare(version, pnpmWarningFixVersion) < 0 {
		// pnpm < 9.7.0 prints warnings to stdout, filter them out
		var filtered []string
		for _, line := range strings.Split(string(output), "\n") {
			if !strings.HasPrefix(line, pnpmWarningPrefix) {
				filtered = append(filtered, line)
			}
		}
		cacheDir = strings.TrimSpace(strings.Join(filtered, "\n"))
	}

	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from pnpm store path")
	}

	// Hard-linking and clone do not work with cache volumes. Select copy mode to avoid spurious warnings.
	return PlanResult{
		AddEnvs: map[string]string{
			pnpmPackageImportMethodKey: pnpmPackageImportMethodValue,
		},
		MountPaths: []string{cacheDir},
	}, nil
}

// PoetryProvider

const poetryLockFile = "poetry.lock"

type PoetryProvider struct{}

func (p PoetryProvider) Name() string {
	return "poetry"
}

func (p PoetryProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("poetry"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath poetry: %w", err)
	}

	if _, err := req.Exec.Stat(poetryLockFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", poetryLockFile, err)
	}

	return true, nil
}

func (p PoetryProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "poetry", "config", "cache-dir")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("poetry config cache-dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from poetry config cache-dir")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}

// PythonProvider

const pythonRequirementsFile = "requirements.txt"

type PythonProvider struct{}

func (p PythonProvider) Name() string {
	return "python"
}

func (p PythonProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("pip"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath pip: %w", err)
	}

	if _, err := req.Exec.Stat(pythonRequirementsFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", pythonRequirementsFile, err)
	}

	return true, nil
}

func (p PythonProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "pip", "cache", "dir")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("pip cache dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from pip cache dir")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}

// RubyProvider

const rubyGemfile = "Gemfile"

type RubyProvider struct{}

func (p RubyProvider) Name() string {
	return "ruby"
}

func (p RubyProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("bundle"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath bundle: %w", err)
	}

	if _, err := req.Exec.Stat(rubyGemfile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", rubyGemfile, err)
	}

	return true, nil
}

func (p RubyProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	return PlanResult{
		MountPaths: []string{
			"./vendor/bundle", // Caches output of `bundle install`
			"./vendor/cache",  // Caches output of `bundle cache` (less common)
		},
	}, nil
}

// RustProvider

const rustCargoToml = "Cargo.toml"

type RustProvider struct{}

func (p RustProvider) Name() string {
	return "rust"
}

func (p RustProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("cargo"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath cargo: %w", err)
	}

	if _, err := req.Exec.Stat(rustCargoToml); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", rustCargoToml, err)
	}

	return true, nil
}

func (p RustProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	// Do not cache the whole ~/.cargo dir as it contains ~/.cargo/bin, where the cargo binary lives.
	return PlanResult{
		MountPaths: []string{
			"~/.cargo/registry",
			"~/.cargo/git",
			"./target",
			"~/.cargo/.global-cache", // Cache cleaning feature uses SQLite file: https://blog.rust-lang.org/2023/12/11/cargo-cache-cleaning.html
		},
	}, nil
}

// SwiftPMProvider

const swiftPackageFile = "Package.swift"

type SwiftPMProvider struct{}

func (p SwiftPMProvider) Name() string {
	return "swiftpm"
}

func (p SwiftPMProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("swift"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath swift: %w", err)
	}

	if _, err := req.Exec.Stat(swiftPackageFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", swiftPackageFile, err)
	}

	return true, nil
}

func (p SwiftPMProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	mountPaths := []string{
		"./.build",
		"~/Library/Caches/org.swift.swiftpm",
		"~/Library/org.swift.swiftpm",
	}

	if !slices.Contains(req.EnabledModes, (XcodeProvider{}).Name()) {
		// Xcode caching already caches all derived data.
		// Cached data lands in the same location, so also restoring with `swiftpm` mode will work.
		mountPaths = append(mountPaths, "~/Library/Developer/Xcode/DerivedData/ModuleCache.noindex")
	}

	return PlanResult{
		MountPaths: mountPaths,
	}, nil
}

// UVProvider

const (
	uvLinkModeKey   = "UV_LINK_MODE"
	uvLinkModeValue = "symlink"
	uvLockFile      = "uv.lock"
)

type UVProvider struct{}

func (p UVProvider) Name() string {
	return "uv"
}

func (p UVProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("uv"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath uv: %w", err)
	}

	if _, err := req.Exec.Stat(uvLockFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", uvLockFile, err)
	}

	return true, nil
}

func (p UVProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "uv", "cache", "dir")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("uv cache dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from uv cache dir")
	}

	// UV defaults to clone (Copy-on-Write) on macOS, and hardlink on Linux and Windows.
	// Neither works with cache volumes, and fall back to `copy`. Select `symlink` to avoid copies.
	return PlanResult{
		AddEnvs: map[string]string{
			uvLinkModeKey: uvLinkModeValue,
		},
		MountPaths: []string{cacheDir},
	}, nil
}

// XcodeProvider

const (
	xcodeCompilationCacheKey   = "COMPILATION_CACHE_ENABLE_CACHING_DEFAULT"
	xcodeCompilationCacheValue = "YES"
	// Consider: `defaults read com.apple.dt.Xcode.plist IDECustomDerivedDataLocation`
	xcodeCachePath  = "~/Library/Developer/Xcode/DerivedData/CompilationCache.noindex"
	xcodeProjSuffix = ".xcodeproj"
)

type XcodeProvider struct{}

func (p XcodeProvider) Name() string {
	return "xcode"
}

func (p XcodeProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("xcodebuild"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath xcodebuild: %w", err)
	}

	entries, err := req.Exec.ReadDir(".")
	if err != nil {
		return false, fmt.Errorf("readdir: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), xcodeProjSuffix) {
			return true, nil
		}
	}

	return false, nil
}

// Experimental: Xcode compilation cache can be huge.
func (p XcodeProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	return PlanResult{
		AddEnvs: map[string]string{
			xcodeCompilationCacheKey: xcodeCompilationCacheValue,
		},
		MountPaths: []string{xcodeCachePath},
	}, nil
}

// YarnProvider

const (
	yarnV1Prefix = "1."
	yarnLockFile = "yarn.lock"
)

type YarnProvider struct{}

func (p YarnProvider) Name() string {
	return "yarn"
}

func (p YarnProvider) Detect(ctx context.Context, req DetectRequest) (bool, error) {
	if _, err := req.Exec.LookPath("yarn"); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("lookpath yarn: %w", err)
	}

	if _, err := req.Exec.Stat(yarnLockFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat %s: %w", yarnLockFile, err)
	}

	return true, nil
}

func (p YarnProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	versionCmd := exec.CommandContext(ctx, "yarn", "--version")
	versionOutput, err := req.Exec.Output(versionCmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("yarn --version: %w", err)
	}
	version := strings.TrimSpace(string(versionOutput))

	// Yarn v1.x uses "yarn cache dir", v2+ uses "yarn config get cacheFolder"
	var cmd *exec.Cmd
	if strings.HasPrefix(version, yarnV1Prefix) {
		cmd = exec.CommandContext(ctx, "yarn", "cache", "dir")
	} else {
		cmd = exec.CommandContext(ctx, "yarn", "config", "get", "cacheFolder")
	}

	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("yarn cache dir: %w", err)
	}

	cacheDir := strings.TrimSpace(string(output))
	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("empty cache dir from yarn")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}
