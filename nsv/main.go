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
	"strings"

	"dagger/nsv/internal/dagger"
)

const (
	NsvGithubRepo = "purpleclay/nsv"
	NsvBaseImage  = "ghcr.io/purpleclay/nsv"
)

// NSV dagger module
type Nsv struct {
	// a custom base image containing an installation of nsv
	// +private
	Base *Container

	// a path to a directory containing the projects source code
	// +private
	Src *Directory
}

// Initializes the NSV dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of nsv
	// +optional
	base *Container,
	// a path to a directory containing the projects source code
	// +required
	src *Directory,
) (*Nsv, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	return &Nsv{Base: base, Src: src}, nil
}

func defaultImage(ctx context.Context) (*Container, error) {
	tag, err := dag.Github().GetLatestRelease(NsvGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From(fmt.Sprintf("%s:%s", NsvBaseImage, tag)), nil
}

// Prints the next semantic version based on the commit history of your repository
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
	// a comma separated list of conventional commit prefixes for triggering a
	// major semantic version increment
	// +optional
	majorPrefixes []string,
	// a comma separated list of conventional commit prefixes for triggering a
	// minor semantic version increment
	// +optional
	minorPrefixes []string,
	// a comma separated list of conventional commit prefixes for triggering a
	// patch semantic version increment
	// +optional
	patchPrefixes []string,
) (string, error) {
	cmd := []string{"next"}
	cmd = append(cmd, formatArgs(majorPrefixes, minorPrefixes, patchPrefixes, pretty, show, paths)...)

	return n.Base.
		WithEnvVariable("TINI_SUBREAPER", "1").
		WithDirectory("/src", n.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Stdout(ctx)
}

func formatArgs(
	majorPrefixes, minorPrefixes, patchPrefixes []string,
	pretty string,
	show bool,
	paths []string,
) []string {
	var args []string

	if show {
		args = append(args, "--show", fmt.Sprintf("--pretty=%s", pretty))
	}

	if len(majorPrefixes) > 0 {
		args = append(args, "--major-prefixes", strings.Join(majorPrefixes, ","))
	}

	if len(minorPrefixes) > 0 {
		args = append(args, "--minor-prefixes", strings.Join(majorPrefixes, ","))
	}

	if len(patchPrefixes) > 0 {
		args = append(args, "--patch-prefixes", strings.Join(majorPrefixes, ","))
	}

	if len(paths) > 0 {
		args = append(args, paths...)
	}

	return args
}

// Tags the next semantic version based on the commit history of your repository
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
	// a custom message for the tag, supports go text templates
	// +optional
	message string,
	// a comma separated list of conventional commit prefixes for triggering a
	// major semantic version increment
	// +optional
	majorPrefixes []string,
	// a comma separated list of conventional commit prefixes for triggering a
	// minor semantic version increment
	// +optional
	minorPrefixes []string,
	// a comma separated list of conventional commit prefixes for triggering a
	// patch semantic version increment
	// +optional
	patchPrefixes []string,
	// an optional passphrase to unlock the GPG private key used for signing the tag
	// +optional
	gpgPassphrase *dagger.Secret,
	// a base64 encoded GPG private key (armored) used for signing the tag
	// +optional
	gpgPrivateKey *dagger.Secret,
) (string, error) {
	cmd := []string{"tag"}
	if message != "" {
		cmd = append(cmd, "--message", message)
	}

	cmd = append(cmd, formatArgs(majorPrefixes, minorPrefixes, patchPrefixes, pretty, show, paths)...)

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
