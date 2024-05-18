// Semantic versioning without any config
//
// NSV (Next Semantic Version) is a convention-based semantic versioning tool that leans on the
// power of conventional commits to make versioning your software a breeze!
//
// There are many semantic versioning tools already out there! But they typically require some
// configuration or custom scripting in your CI system to make them work. No one likes managing config;
// it is error-prone, and the slightest tweak ultimately triggers a cascade of change across your projects.
//
// Step in NSV. Designed to make intelligent semantic versioning decisions about your project without needing a
// config file. Entirely convention-based, you can adapt your workflow from within your commit message.
//
// The power is at your fingertips.
package main

import (
	"context"
	"fmt"

	"dagger/nsv/internal/dagger"
)

// NSV dagger module
type Nsv struct {
	// Base is the image used by all nsv dagger functions
	// +private
	Base *Container

	// Src is a directory that contains the projects source code
	// +private
	Src *Directory
}

// New initializes the NSV dagger module
func New(
	// a path to a directory containing the source code
	// +required
	src *Directory,
) *Nsv {
	return &Nsv{Base: base(), Src: src}
}

func base() *Container {
	return dag.Container().
		From("ghcr.io/purpleclay/nsv:v0.7.0")
}

// Prints the next semantic version based on the commit history of your repository
//
// Examples:
//
// # Print the next semantic version
// $ dagger call next
//
// # Print the next semantic version for multiple monorepo sub-projects
// $ dagger call next --paths project1,project2
//
// # Print the next semantic version and show how the version was calculated
// $ dagger call next --show
func (n *Nsv) Next(
	ctx context.Context,
	// a list of relative paths of projects to analyze
	// +optional
	paths []string,
	// show how the next semantic version was calculated
	// +optional
	show bool,
	// pretty-print the output of the next semantic version in a given format.
	// Supported formats are (full, compact). Must be used in conjunction with
	// the show flag
	// +optional
	// +default="full"
	pretty string,
) (string, error) {
	cmd := []string{"next"}
	if show {
		cmd = append(cmd, "--show", fmt.Sprintf("--pretty=%s", pretty))
	}

	if len(paths) > 0 {
		cmd = append(cmd, paths...)
	}

	return n.Base.
		WithEnvVariable("TINI_SUBREAPER", "1").
		WithDirectory("/src", n.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Stdout(ctx)
}

// Tags the next semantic version based on the commit history of your repository
//
// Examples:
//
// # Tag the next semantic version
// $ dagger call tag
//
// # Tag the next semantic version for multiple monorepo sub-projects
// $ dagger call tag --paths project1,project2
//
// # Tag the next semantic version and show how the version was calculated
// $ dagger call tag --show
func (n *Nsv) Tag(
	ctx context.Context,
	// a list of relative paths of projects to analyze
	// +optional
	paths []string,
	// show how the next semantic version was calculated
	// +optional
	show bool,
	// pretty-print the output of the next semantic version in a given format.
	// Supported formats are (full, compact). Must be used in conjunction with
	// the show flag
	// +optional
	// +default="full"
	pretty string,
	// an optional passphrase to unlock the GPG private key used for signing the tag
	// +optional
	gpgPassphrase *dagger.Secret,
	// a base64 encoded GPG private key (armored) used for signing the tag
	// +optional
	gpgPrivateKey *dagger.Secret,
) (string, error) {
	cmd := []string{"tag"}
	if show {
		cmd = append(cmd, "--show", fmt.Sprintf("--pretty=%s", pretty))
	}

	if len(paths) > 0 {
		cmd = append(cmd, paths...)
	}

	ctr := n.Base
	if gpgPrivateKey != nil {
		ctr = ctr.WithSecretVariable("GPG_PRIVATE_KEY", gpgPrivateKey).
			WithEnvVariable("GPG_TRUST_LEVEL", "5")
	}

	if gpgPassphrase != nil {
		ctr = ctr.WithSecretVariable("GPG_PASSPHRASE", gpgPassphrase)
	}

	return ctr.
		WithEnvVariable("TINI_SUBREAPER", "1").
		WithDirectory("/src", n.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Stdout(ctx)
}
