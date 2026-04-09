package helm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"helm.sh/helm/v4/pkg/action"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/cli"
	"helm.sh/helm/v4/pkg/registry"
	releasepkg "helm.sh/helm/v4/pkg/release"
	helmrelease "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage/driver"
)

// ActionConfigGetter returns a Helm action configuration for a namespace.
type ActionConfigGetter interface {
	GetActionConfigE(namespace string) (*action.Configuration, error)
}

// Service runs Helm release lifecycle operations.
type Service struct {
	actionConfigGetter ActionConfigGetter
	registryClient     *registry.Client
	envSettings        *cli.EnvSettings
}

// Spec is the plain Go input for release operations.
type Spec struct {
	ChartRef  string
	Name      string
	Namespace string
	Version   string
	Values    map[string]any
	Timeout   time.Duration
}

// NewService creates a new release service.
func NewService(actionConfigGetter ActionConfigGetter, registryClient *registry.Client, envSettings *cli.EnvSettings) *Service {
	return &Service{
		actionConfigGetter: actionConfigGetter,
		registryClient:     registryClient,
		envSettings:        envSettings,
	}
}

// RenderInstall renders the desired release using a dry-run install.
func (s *Service) RenderInstall(ctx context.Context, spec Spec) (*helmrelease.Release, []string, error) {
	act, loadedChart, warnings, err := s.prepareInstall(spec)
	if err != nil {
		return nil, nil, err
	}

	act.DryRunStrategy = action.DryRunClient

	rel, err := act.RunWithContext(ctx, loadedChart, spec.Values)
	if err != nil {
		return nil, nil, fmt.Errorf("template helm chart: %w", err)
	}

	converted, err := releaserToRelease(rel)
	if err != nil {
		return nil, nil, err
	}

	return converted, warnings, nil
}

// Install installs a Helm release.
func (s *Service) Install(ctx context.Context, spec Spec) (*helmrelease.Release, []string, error) {
	act, loadedChart, warnings, err := s.prepareInstall(spec)
	if err != nil {
		return nil, nil, err
	}

	rel, err := act.RunWithContext(ctx, loadedChart, spec.Values)
	if err != nil {
		return nil, nil, fmt.Errorf("install helm release: %w", err)
	}

	converted, err := releaserToRelease(rel)
	if err != nil {
		return nil, nil, err
	}

	return converted, warnings, nil
}

// Get fetches an installed Helm release.
func (s *Service) Get(spec Spec) (*helmrelease.Release, bool, error) {
	actionConfig, err := s.actionConfigGetter.GetActionConfigE(spec.Namespace)
	if err != nil {
		return nil, false, err
	}

	getAction := action.NewGet(actionConfig)
	rel, err := getAction.Run(spec.Name)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("read helm release: %w", err)
	}

	converted, err := releaserToRelease(rel)
	if err != nil {
		return nil, false, err
	}

	return converted, true, nil
}

// PlanUpgrade renders the upgraded release using Helm dry-run semantics.
func (s *Service) PlanUpgrade(ctx context.Context, spec Spec) (*helmrelease.Release, []string, error) {
	act, loadedChart, warnings, err := s.prepareUpgrade(spec)
	if err != nil {
		return nil, nil, err
	}

	act.DryRunStrategy = action.DryRunClient

	rel, err := act.RunWithContext(ctx, spec.Name, loadedChart, spec.Values)
	if err != nil {
		return nil, nil, fmt.Errorf("plan helm release update: %w", err)
	}

	converted, err := releaserToRelease(rel)
	if err != nil {
		return nil, nil, err
	}

	return converted, warnings, nil
}

// Upgrade upgrades an existing Helm release or installs it if it is missing.
func (s *Service) Upgrade(ctx context.Context, spec Spec) (*helmrelease.Release, []string, error) {
	act, loadedChart, warnings, err := s.prepareUpgrade(spec)
	if err != nil {
		return nil, nil, err
	}

	rel, err := act.RunWithContext(ctx, spec.Name, loadedChart, spec.Values)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return s.Install(ctx, spec)
		}

		return nil, nil, fmt.Errorf("update helm release: %w", err)
	}

	converted, err := releaserToRelease(rel)
	if err != nil {
		return nil, nil, err
	}

	return converted, warnings, nil
}

// Uninstall uninstalls a Helm release.
func (s *Service) Uninstall(spec Spec) error {
	actionConfig, err := s.actionConfigGetter.GetActionConfigE(spec.Namespace)
	if err != nil {
		return err
	}

	act := action.NewUninstall(actionConfig)
	act.IgnoreNotFound = true
	act.KeepHistory = false
	act.Timeout = spec.Timeout

	if _, err := act.Run(spec.Name); err != nil {
		return fmt.Errorf("uninstall helm release: %w", err)
	}

	return nil
}

func (s *Service) prepareInstall(spec Spec) (*action.Install, *chartv2.Chart, []string, error) {
	actionConfig, err := s.actionConfigGetter.GetActionConfigE(spec.Namespace)
	if err != nil {
		return nil, nil, nil, err
	}

	loadedChart, warnings, err := s.loadChart(spec)
	if err != nil {
		return nil, nil, nil, err
	}

	act := action.NewInstall(actionConfig)
	act.SetRegistryClient(s.registryClient)
	act.SkipCRDs = true
	act.Timeout = spec.Timeout
	act.ReleaseName = spec.Name
	act.Namespace = spec.Namespace
	act.Version = spec.Version
	act.ChartPathOptions.Version = act.Version

	return act, loadedChart, warnings, nil
}

func (s *Service) prepareUpgrade(spec Spec) (*action.Upgrade, *chartv2.Chart, []string, error) {
	actionConfig, err := s.actionConfigGetter.GetActionConfigE(spec.Namespace)
	if err != nil {
		return nil, nil, nil, err
	}

	loadedChart, warnings, err := s.loadChart(spec)
	if err != nil {
		return nil, nil, nil, err
	}

	act := action.NewUpgrade(actionConfig)
	act.SetRegistryClient(s.registryClient)
	act.Namespace = spec.Namespace
	act.Timeout = spec.Timeout
	act.ResetValues = true
	act.SkipCRDs = true
	act.ChartPathOptions.Version = spec.Version

	return act, loadedChart, warnings, nil
}

func (s *Service) loadChart(spec Spec) (*chartv2.Chart, []string, error) {
	result, err := LoadInstallable(s.envSettings, spec.ChartRef, spec.Version)
	if err != nil {
		return nil, nil, err
	}

	return result.Chart, result.Warnings, nil
}

func releaserToRelease(rel releasepkg.Releaser) (*helmrelease.Release, error) {
	if release, ok := rel.(*helmrelease.Release); ok {
		return release, nil
	}

	if release, ok := rel.(helmrelease.Release); ok {
		return &release, nil
	}

	return nil, fmt.Errorf("expected *release.Release, got %T", rel)
}
