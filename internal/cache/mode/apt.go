package mode

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

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
