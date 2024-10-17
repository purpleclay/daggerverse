// A generated module for Kubeconform functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/kubeconform/internal/dagger"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	KubeconformGithubRepo = "yannh/kubeconform"
	KubeconformBaseImage  = "ghcr.io/yannh/kubeconform"
	KubeconformWorkDir    = "/work"
)

// Kubeconform dagger module
type Kubeconform struct {
	// +private
	Base *dagger.Container
}

// Initializes the Kubeconform dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of kubeconform
	// +optional
	base *dagger.Container,
) (*Kubeconform, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"kubeconform", "-v"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	return &Kubeconform{Base: base}, err
}

func defaultImage(ctx context.Context) (*dagger.Container, error) {
	tag, err := dag.Github().GetLatestRelease(KubeconformGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From(fmt.Sprintf("%s:v%s", KubeconformBaseImage, tag[1:])), nil
}

// TODO
func (m *Kubeconform) Validate(
	ctx context.Context,
	// a path to a directory containing Kubernetes manifests for validation
	// +optional
	dirs []*dagger.Directory,
	// skip files with missing schemas instead of failing
	// +optional
	ignoreMissingSchemas bool,
	// disable verification of the server's SSL certificate
	// +optional
	insecureSkipTlsVerify bool,
	// the version of kubernertes to validate against, e.g. 1.31.0
	// +optional
	// +default="master"
	kubernetesVersion string,
	// the number of goroutines to run concurrently during validation
	// +optional
	// +default=4
	goroutines int,
	// a path to a Kubernetes manifest file for validation
	// +optional
	files []*dagger.File,
	// a comma-separated list of kinds or GVKs to reject
	// +optional
	reject []string,
	// override the schema search location path
	// +optional
	schemaLocation string,
	// print results for all resources (verbose)
	// +optional
	show bool,
	// a comma-separated list of kinds or GVKs to ignore
	// +optional
	skip []string,
	// disallow additional properties not in schema or duplicated keys
	// +optional
	strict bool,
	// print a summary at the end
	// +optional
	summary bool,
) (string, error) {
	// TODO: how to determine the entrypoint to execute
	cmd := []string{}
	if ignoreMissingSchemas {
		cmd = append(cmd, "-ignore-missing-schemas")
	}

	if insecureSkipTlsVerify {
		cmd = append(cmd, "-insecure-skip-tls-verify")
	}

	if kubernetesVersion != "master" {
		cmd = append(cmd, "-kubernetes-version", kubernetesVersion)
	}

	if goroutines != 4 && goroutines > 0 {
		cmd = append(cmd, "-n", strconv.Itoa(int(goroutines)))
	}

	if len(reject) > 0 {
		cmd = append(cmd, "-reject", strings.Join(reject, ","))
	}

	if schemaLocation != "" {
		cmd = append(cmd, "-schema-location", schemaLocation)
	}

	if len(skip) > 0 {
		cmd = append(cmd, "-skip", strings.Join(skip, ","))
	}

	if strict {
		cmd = append(cmd, "-strict")
	}

	if summary {
		cmd = append(cmd, "-summary")
	}

	if show {
		cmd = append(cmd, "-verbose")
	}

	ctr := m.Base.WithWorkdir(KubeconformWorkDir)

	counter := 1
	for _, file := range files {
		fname, err := file.Name(ctx)
		if err != nil {
			return "", err
		}

		copyTo := filepath.Join(fmt.Sprintf("%03d", counter), fname)
		cmd = append(cmd, copyTo)

		ctr = ctr.WithFile(copyTo, file, dagger.ContainerWithFileOpts{Permissions: 0o644})
		counter++
	}

	for _, dir := range dirs {
		copyTo := filepath.Join(fmt.Sprintf("%03d", counter), fname)

		ctr = ctr.WithDirectory(fmt.Sprintf("%03d", counter), dir)
		cmd = append(cmd, copyTo)
	}

	return ctr.
		WithExec(cmd, dagger.ContainerWithExecOpts{UseEntrypoint: true}).
		Stdout(ctx)
}
