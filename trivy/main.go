package main

import (
	"context"
	"fmt"
	"strconv"
)

const (
	TrivyGithubRepo = "aquasecurity/trivy"
	TrivyBaseImage  = "ghcr.io/aquasecurity/trivy"
)

// Trivy dagger module
type Trivy struct {
	// Base is the image used by all trivy dagger functions
	// +private
	Base *Container
}

// --format template --template "@contrib/junit.tpl"

// New initializes the trivy dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of trivy
	// +optional
	base *Container) (*Trivy, error) {

	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"version"}).Sync(ctx); err != nil {
			return nil, err
		}

		base = base.WithMountedCache("/root/.cache/trivy", dag.CacheVolume("trivydb"))
	}

	return &Trivy{Base: base}, err
}

func defaultImage(ctx context.Context) (*Container, error) {
	tag, err := dag.Github().GetLatestRelease("aquasecurity/trivy").Tag(ctx)
	if err != nil {
		return nil, err
	}

	// Trim the v prefix from the tag
	return dag.Container().
		From(fmt.Sprintf("%s:%s", TrivyBaseImage, tag[1:])).
		WithMountedCache("/root/.cache/trivy", dag.CacheVolume("trivydb")), nil
}

// Scan a published (or remote) image for any vulnerabilities
func (t *Trivy) Image(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected
	// +optional
	// +default=0
	exitCode int,
	// the type of format to use when generating the compliance report
	// +optional
	// +default="table"
	format string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the reference to an image within a repository
	// +required
	ref string,
	// the types of scanner to execute
	// +optional
	// +default="vuln,secret"
	scanners string,
	// the severity of security issues to detect
	// +optional
	// +default="UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL"
	severity string,
	// a custom go template to use when generating the compliance report
	// +optional
	template string,
	// the types of vulnerabilities to scan for
	// +optional
	// +default="os,library"
	vulnType string) (string, error) {
	cmd := []string{
		"image",
		ref,
		"--scanners",
		scanners,
		"--severity",
		severity,
		"--vuln-type",
		vulnType,
		"--exit-code",
		strconv.Itoa(exitCode),
		"--format",
		format,
	}
	if ignoreUnfixed {
		cmd = append(cmd, "--ignore-unfixed")
	}
	if template != "" {
		cmd = append(cmd, "--template", template)
	}

	return t.Base.WithExec(cmd).Stdout(ctx)
}

// Scan a locally exported image for any vulnerabilities
func (t *Trivy) ImageLocal(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected
	// +optional
	// +default=0
	exitCode int,
	// the type of format to use when generating the compliance report
	// +optional
	// +default="table"
	format string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the path to an exported image tar
	// +required
	ref *File,
	// the types of scanner to execute
	// +optional
	// +default="vuln,secret"
	scanners string,
	// the severity of security issues to detect
	// +optional
	// +default="UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL"
	severity string,
	// a custom go template to use when generating the compliance report
	// +optional
	template string,
	// the types of vulnerabilities to scan for
	// +optional
	// +default="os,library"
	vulnType string) (string, error) {
	cmd := []string{
		"image",
		"--input",
		"/scan/image.tar",
		"--scanners",
		scanners,
		"--severity",
		severity,
		"--vuln-type",
		vulnType,
		"--exit-code",
		strconv.Itoa(exitCode),
		"--format",
		format,
	}
	if ignoreUnfixed {
		cmd = append(cmd, "--ignore-unfixed")
	}
	if template != "" {
		cmd = append(cmd, "--template", template)
	}

	return t.Base.
		WithMountedFile("/scan/image.tar", ref).
		WithExec(cmd).
		Stdout(ctx)
}

// Scan a filesystem for any vulnerabilities
func (t *Trivy) Filesystem(
	ctx context.Context,
	// the path to directory to scan
	// +required
	dir *Directory,
	// the returned exit code when vulnerabilities are detected
	// +optional
	// +default=0
	exitCode int,
	// the type of format to use when generating the compliance report
	// +optional
	// +default="table"
	format string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the types of scanner to execute
	// +optional
	// +default="vuln,secret"
	scanners string,
	// the severity of security issues to detect
	// +optional
	// +default="UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL"
	severity string,
	// a custom go template to use when generating the compliance report
	// +optional
	template string,
	// the types of vulnerabilities to scan for
	// +optional
	// +default="os,library"
	vulnType string) (string, error) {
	cmd := []string{
		"filesystem",
		".",
		"--scanners",
		scanners,
		"--severity",
		severity,
		"--vuln-type",
		vulnType,
		"--exit-code",
		strconv.Itoa(exitCode),
		"--format",
		format,
	}
	if ignoreUnfixed {
		cmd = append(cmd, "--ignore-unfixed")
	}
	if template != "" {
		cmd = append(cmd, "--template", template)
	}

	return t.Base.
		WithDirectory("/scan", dir).
		WithWorkdir("/scan").
		WithExec(cmd).
		Stdout(ctx)
}
