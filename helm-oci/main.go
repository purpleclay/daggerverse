// A lightweight wrapper around Helm OCI.
package main

import (
	"context"
	"fmt"
	"strings"
)

const (
	HelmGithubRepo       = "helm/helm"
	HelmBaseImage        = "alpine/helm"
	HelmRepositoryConfig = "/root/.config/helm/registry/config.json"
)

// Helm OCI dagger module
type HelmOci struct {
	// +private
	Base *Container
}

// Initializes the Helm OCI dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of helm
	// +optional
	base *Container,
) (*HelmOci, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	base = base.WithUser("root").
		WithoutEnvVariable("HELM_HOME").
		WithoutEnvVariable("HELM_REGISTRY_CONFIG")

	return &HelmOci{Base: base}, err
}

func defaultImage(ctx context.Context) (*Container, error) {
	tag, err := dag.Github().GetLatestRelease(HelmGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From(fmt.Sprintf("%s:%s", HelmBaseImage, tag[1:])), nil
}

// Packages a Helm chart and publishes it to an OCI registry
func (m *HelmOci) PackagePush(
	ctx context.Context,
	// a path to the directory containing the Chart.yaml file
	// +required
	chart *Directory,
	// the OCI registry to publish the chart to, should include full path without chart name
	// +required
	registry string,
	// the username for authenticating with the registry
	// +required
	username string,
	// the password for authenticating with the registry
	// +required
	password *Secret,
) (string, error) {

	ctr := m.Base.
		WithMountedDirectory("/work", chart).
		WithWorkdir("/work")

	tgz, err := ctr.WithExec([]string{"package", "."}).Stdout(ctx)
	if err != nil {
		return "", err
	}
	tgz = tgz[strings.LastIndex(tgz, "/")+1 : len(tgz)-1]

	// Extract the registry host needed for logging in
	registry = strings.TrimPrefix(registry, "oci://")

	idx := strings.Index(registry, "/")
	if idx == -1 {
		return "", fmt.Errorf("malformed registry, could not extract host")
	}
	registryHost := registry[:idx]

	// https://github.com/dagger/dagger/issues/7274
	helmAuth := dag.RegistryConfig().WithRegistryAuth(registryHost, username, password).Secret()

	return ctr.
		WithMountedSecret(HelmRepositoryConfig, helmAuth).
		WithExec([]string{"push", tgz, fmt.Sprintf("oci://%s", registry)}).
		Stdout(ctx)
}

// Lints a Helm chart
func (m *HelmOci) Lint(
	ctx context.Context,
	// a path to the directory containing the Chart.yaml file
	// +required
	chart *Directory,
	// fail on any linting errors by returning a non zero exit code
	// +optional
	strict bool,
	// print only warnings and errors
	// +optional
	quiet bool) (string, error) {

	cmd := []string{"lint", "."}

	if strict {
		cmd = append(cmd, "--strict")
	}

	if quiet {
		cmd = append(cmd, "--quiet")
	}

	return m.Base.
		WithMountedDirectory("/work", chart).
		WithWorkdir("/work").
		WithExec(cmd).
		Stdout(ctx)
}
