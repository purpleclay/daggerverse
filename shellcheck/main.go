// ShellCheck, a static analysis tool for your shell scripts
//
// The goals of ShellCheck are:
//
// - To point out and clarify typical beginner's syntax issues that cause a shell to give cryptic error messages.
// - To point out and clarify typical intermediate level semantic problems that cause a shell to behave strangely and counter-intuitively.
// - To point out subtle caveats, corner cases and pitfalls that may cause an advanced user's otherwise working script to fail under future circumstances.
package main

import (
	"context"
	"dagger/shellcheck/internal/dagger"
	"fmt"
	"strings"
)

const (
	ShellcheckGithubRepo = "koalaman/shellcheck"
	ShellcheckBaseImage  = "koalaman/shellcheck-alpine"
	WorkingDir           = "/work"
)

// ShellCheck dagger module
type Shellcheck struct {
	// a custom base image containing an installation of shellcheck
	// +private
	Base *dagger.Container
}

// Initializes the ShellCheck dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of shellcheck
	// +optional
	base *dagger.Container,
) (*Shellcheck, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"shellcheck", "--version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	return &Shellcheck{Base: base}, err
}

func defaultImage(ctx context.Context) (*dagger.Container, error) {
	tag, err := dag.Github().GetLatestRelease(ShellcheckGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From(fmt.Sprintf("%s:%s", ShellcheckBaseImage, tag)), nil
}

// Checks shell scripts for syntactic and semantic issues that may otherwise be difficult
// to identify
func (m *Shellcheck) Check(
	ctx context.Context,
	// the output format of the shellcheck report
	// (checkstyle, diff, gcc, json, json1, quiet, tty)
	// +optional
	format string,
	// a list of paths for checking
	// +optional
	// +default=["*.sh"]
	paths []string,
	// the minimum severity of errors to consider when checking scripts
	// (error, warning, info, style)
	// +optional
	severity string,
	// the type of shell dialect to check against (sh, bash, dash, ksh, busybox)
	// +optional
	shell string,
	// a path to a directory containing scripts to scan, this can be a project root
	// +required
	src *dagger.Directory,
) (string, error) {
	cmd := []string{"shellcheck"}
	if format != "" {
		cmd = append(cmd, "--format", format)
	}

	if severity != "" {
		cmd = append(cmd, "--severity", severity)
	}

	if shell != "" {
		cmd = append(cmd, "--shell", shell)
	}

	for _, toCheck := range paths {
		cmd = append(cmd, toCheck)
	}

	return m.Base.
		WithDirectory(WorkingDir, src).
		WithWorkdir(WorkingDir).
		WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).
		Stdout(ctx)
}
