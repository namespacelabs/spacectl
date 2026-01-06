//go:generate moq -out mode_mock.go . Executor ModeProvider
package mode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"
)

func DefaultModes() Modes {
	return Modes{
		AptProvider{},
		GoProvider{},
		GolangCILintProvider{},
	}
}

type Modes []ModeProvider

// Names returns all mode names that can be used.
func (modes Modes) Names() []string {
	avail := make([]string, 0, len(modes))
	for _, mode := range modes {
		avail = append(avail, mode.Name())
	}
	slices.Sort(avail)
	return avail
}

// Filter reduces modes down to those specified in the from slice.
func (modes Modes) Filter(include []string) (Modes, error) {
	if len(modes) == 0 {
		return nil, nil
	}

	available := make(map[string]ModeProvider, len(modes))
	for _, mode := range modes {
		available[mode.Name()] = mode
	}

	filtered := make(Modes, 0, len(include))
	for _, inc := range include {
		mode, ok := available[inc]
		if !ok {
			return nil, fmt.Errorf("unknown mode: %s", inc)
		}
		filtered = append(filtered, mode)
	}

	return filtered, nil
}

// Detect runs detection for all modes in parallel and returns those that were detected.
func (modes Modes) Detect(ctx context.Context, req DetectRequest) (Modes, error) {
	if req.Exec == nil {
		req.Exec = DefaultExecutor{}
	}

	var m sync.Mutex
	filtered := make(Modes, 0, len(modes))

	eg, ctx := errgroup.WithContext(ctx)
	for _, mode := range modes {
		eg.Go(func() error {
			detected, err := mode.Detect(ctx, req)
			if err != nil {
				return fmt.Errorf("detecting %s: %w", mode.Name(), err)
			}
			if detected {
				m.Lock()
				defer m.Unlock()
				filtered = append(filtered, mode)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return filtered, nil
}

// Plan runs planning for all modes in parallel and returns their results.
func (modes Modes) Plan(ctx context.Context, req PlanRequest) (map[string]PlanResult, error) {
	req.enabledModes = modes.Names()
	if req.Exec == nil {
		req.Exec = DefaultExecutor{}
	}

	var m sync.Mutex
	plans := make(map[string]PlanResult, len(modes))

	eg, ctx := errgroup.WithContext(ctx)
	for _, mode := range modes {
		eg.Go(func() error {
			result, err := mode.Plan(ctx, req)
			if err != nil {
				return fmt.Errorf("planning %s: %w", mode.Name(), err)
			}

			m.Lock()
			plans[mode.Name()] = result
			m.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return plans, nil
}

type ModeProvider interface {
	Name() string
	Detect(ctx context.Context, req DetectRequest) (bool, error)
	Plan(ctx context.Context, req PlanRequest) (PlanResult, error)
}

type DetectRequest struct {
	Exec Executor
}

type PlanRequest struct {
	Exec         Executor
	enabledModes []string
}

type PlanResult struct {
	AddEnvs     map[string]string
	MountPaths  []string
	RemovePaths []string
}

type Executor interface {
	LookPath(file string) (string, error)
	Output(*exec.Cmd) ([]byte, error)
	Stat(name string) (os.FileInfo, error)
}

type DefaultExecutor struct{}

func (e DefaultExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (e DefaultExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func (e DefaultExecutor) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
