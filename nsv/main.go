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

// Supported log levels
type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
	Fatal LogLevel = "fatal"
)

// NSV dagger module
type Nsv struct {
	// a custom base image containing an installation of nsv
	// +private
	Base *dagger.Container

	// a path to a directory containing the projects source code
	// +private
	Src *dagger.Directory
}

// Initializes the NSV dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of nsv
	// +optional
	base *dagger.Container,
	// a path to a directory containing the projects source code
	// +required
	src *dagger.Directory,
	// silence all logging within nsv
	// +optional
	noLog bool,
	// the level of logging when printing to stderr
	// +default="info"
	logLevel LogLevel,
) (*Nsv, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	if noLog {
		base = base.WithEnvVariable("NO_LOG", "true")
	}

	base = base.WithEnvVariable("LOG_LEVEL", string(logLevel))
	return &Nsv{Base: base, Src: src}, nil
}

func defaultImage(ctx context.Context) (*dagger.Container, error) {
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
	// provide a go template for changing the default version format
	// +optional
	format string,
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
	// a list of relative paths of projects to analyze
	// +optional
	paths []string,
	// pretty-print the output of the next semantic version in a given format.
	// Supported formats are (full, compact). Must be used in conjunction with
	// the show flag
	// +optional
	// +default="full"
	pretty string,
	// show how the next semantic version was calculated
	// +optional
	show bool,
) (string, error) {
	cmd := []string{"nsv", "next"}
	cmd = append(cmd, formatArgs(format, majorPrefixes, minorPrefixes, patchPrefixes, pretty, show, paths)...)

	return n.Base.
		WithDirectory("/src", n.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Stdout(ctx)
}

func formatArgs(
	format string,
	majorPrefixes, minorPrefixes, patchPrefixes []string,
	pretty string,
	show bool,
	paths []string,
) []string {
	var args []string

	if format != "" {
		args = append(args, "--format", format)
	}

	if show {
		args = append(args, "--show", fmt.Sprintf("--pretty=%s", pretty))
	}

	if len(majorPrefixes) > 0 {
		args = append(args, "--major-prefixes", strings.Join(majorPrefixes, ","))
	}

	if len(minorPrefixes) > 0 {
		args = append(args, "--minor-prefixes", strings.Join(minorPrefixes, ","))
	}

	if len(patchPrefixes) > 0 {
		args = append(args, "--patch-prefixes", strings.Join(patchPrefixes, ","))
	}

	if len(paths) > 0 {
		args = append(args, paths...)
	}

	return args
}

// Tags the next semantic version based on the commit history of your repository.
// Includes experimental support for patching files through a custom hook
func (n *Nsv) Tag(
	ctx context.Context,
	// provide a go template for changing the default version format
	// +optional
	format string,
	// an optional passphrase to unlock the GPG private key used for signing the tag
	// +optional
	gpgPassphrase *dagger.Secret,
	// a base64 encoded GPG private key (armored) used for signing the tag
	// +optional
	gpgPrivateKey *dagger.Secret,
	// a user-defined hook that will be executed before the repository is tagged
	// with the next semantic version. Can be inline shell or a path to a script
	// +optional
	hook string,
	// a comma separated list of conventional commit prefixes for triggering a
	// major semantic version increment
	// +optional
	majorPrefixes []string,
	// a custom message for the tag, supports go text templates
	// +optional
	message string,
	// a comma separated list of conventional commit prefixes for triggering a
	// minor semantic version increment
	// +optional
	minorPrefixes []string,
	// a comma separated list of conventional commit prefixes for triggering a
	// patch semantic version increment
	// +optional
	patchPrefixes []string,
	// a list of relative paths of projects to analyze
	// +optional
	paths []string,
	// pretty-print the output of the next semantic version in a given format.
	// Supported formats are (full, compact). Must be used in conjunction with
	// the show flag
	// +optional
	// +default="full"
	pretty string,
	// show how the next semantic version was calculated
	// +optional
	show bool,
) (string, error) {
	cmd := []string{"nsv", "tag"}
	if message != "" {
		cmd = append(cmd, "--message", message)
	}

	if hook != "" {
		cmd = append(cmd, "--hook", hook)
	}

	cmd = append(cmd, formatArgs(format, majorPrefixes, minorPrefixes, patchPrefixes, pretty, show, paths)...)

	ctr := n.Base
	if gpgPrivateKey != nil {
		ctr = ctr.WithSecretVariable("GPG_PRIVATE_KEY", gpgPrivateKey).
			WithEnvVariable("GPG_TRUST_LEVEL", "5")
	}

	if gpgPassphrase != nil {
		ctr = ctr.WithSecretVariable("GPG_PASSPHRASE", gpgPassphrase)
	}

	return ctr.
		WithDirectory("/src", n.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Stdout(ctx)
}
