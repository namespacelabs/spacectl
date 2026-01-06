package mode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	goCacheKey     = "GOCACHE"
	goModeCacheKey = "GOMODCACHE"
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
	return true, nil
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
	return true, nil
}

func (p GolangCILintProvider) Plan(ctx context.Context, req PlanRequest) (PlanResult, error) {
	cmd := exec.CommandContext(ctx, "golangci-lint", "cache", "status")
	output, err := req.Exec.Output(cmd)
	if err != nil {
		return PlanResult{}, fmt.Errorf("golangci-lint cache status: %w", err)
	}

	var cacheDir string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		prefix := "dir:"

		if strings.HasPrefix(strings.ToLower(line), prefix) {
			cacheDir = strings.TrimSpace(line[len(prefix):])
			break
		}
	}
	if scanner.Err() != nil {
		return PlanResult{}, fmt.Errorf("scanning golangci-lint output: %w", scanner.Err())
	}

	if cacheDir == "" {
		return PlanResult{}, fmt.Errorf("cache dir not found in golangci-lint output")
	}

	return PlanResult{
		MountPaths: []string{cacheDir},
	}, nil
}
