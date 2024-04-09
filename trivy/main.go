package main

import (
	"context"
	"fmt"
	"strconv"
)

const (
	TrivyGithubRepo = "aquasecurity/trivy"
	TrivyBaseImage  = "ghcr.io/aquasecurity/trivy"
	TrivyWorkDir    = "scan"
)

// Trivy dagger module
type Trivy struct {
	// Base is the image used by all trivy dagger functions
	// +private
	Base *Container
	// Identified whether the experimental YAML format for the
	// ignore file has been provided. Once this is stable, it
	// will be loaded automatically
	// +private
	IgnoreFile string
}

type scanArgs struct {
	ExitCode      int
	Format        string
	IgnoreFile    string
	IgnoreUnfixed bool
	Scanners      string
	Severity      string
	Template      string
	VulnType      string
}

func (a scanArgs) Args() []string {
	args := []string{}
	if a.ExitCode != 0 {
		args = append(args, "--exit-code", strconv.Itoa(a.ExitCode))
	}

	if a.Format != "" {
		args = append(args, "--format", a.Format)
	}

	if a.IgnoreFile != "" {
		args = append(args, "--ignorefile", a.IgnoreFile)
	}

	if a.IgnoreUnfixed {
		args = append(args, "--ignore-unfixed")
	}

	if a.Scanners != "" {
		args = append(args, "--scanners", a.Scanners)
	}

	if a.Severity != "" {
		args = append(args, "--severity", a.Severity)
	}

	if a.Template != "" {
		args = append(args, "--template", a.Template)
	}

	if a.VulnType != "" {
		args = append(args, "--vuln-type", a.VulnType)
	}

	return args
}

// New initializes the trivy dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of trivy
	// +optional
	base *Container,
	// a trivy configuration file, https://aquasecurity.github.io/trivy/latest/docs/configuration/
	// Will be mounted as trivy.yaml
	// +optional
	cfg *File,
	// a trivy ignore file for configuring supressions,
	// https://aquasecurity.github.io/trivy/latest/docs/configuration/filtering/#suppression.
	// Will be mounted as either .trivyignore or .trivyignore.yaml
	// +optional
	ignoreFile *File,
) (*Trivy, error) {

	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	base = base.WithMountedCache("/root/.cache/trivy", dag.CacheVolume("trivydb")).
		WithWorkdir(TrivyWorkDir)

	if cfg != nil {
		base = base.WithMountedFile("trivy.yaml", cfg)
	}

	var ignoreFilePath string
	if ignoreFile != nil {
		name, err := ignoreFile.Name(ctx)
		if err != nil {
			return nil, err
		}

		switch name {
		case ".trivyignore.yml", ".trivyignore.yaml":
			ignoreFilePath = name
			fallthrough
		case ".trivyignore":
			base = base.WithMountedFile(name, ignoreFile)
		}
	}

	return &Trivy{Base: base, IgnoreFile: ignoreFilePath}, err
}

func defaultImage(ctx context.Context) (*Container, error) {
	tag, err := dag.Github().GetLatestRelease("aquasecurity/trivy").Tag(ctx)
	if err != nil {
		return nil, err
	}

	// Trim the v prefix from the tag
	return dag.Container().
		From(fmt.Sprintf("%s:%s", TrivyBaseImage, tag[1:])), nil
}

// Scan a published (or remote) image for any vulnerabilities
func (t *Trivy) Image(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected (0)
	// +optional
	exitCode int,
	// the type of format to use when generating the compliance report (table)
	// +optional
	format string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the reference to an image within a repository
	// +required
	ref string,
	// the types of scanner to execute (vuln,secret)
	// +optional
	scanners string,
	// the severity of security issues to detect (UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL)
	// +optional
	severity string,
	// a custom go template to use when generating the compliance report
	// +optional
	template string,
	// the types of vulnerabilities to scan for (os,library)
	// +optional
	vulnType string) (string, error) {
	cmd := []string{"image", ref}

	sargs := scanArgs{
		ExitCode:      exitCode,
		Format:        format,
		IgnoreFile:    t.IgnoreFile,
		IgnoreUnfixed: ignoreUnfixed,
		Scanners:      scanners,
		Severity:      severity,
		Template:      template,
		VulnType:      vulnType,
	}
	cmd = append(cmd, sargs.Args()...)

	return t.Base.WithExec(cmd).Stdout(ctx)
}

// Scan a locally exported image for any vulnerabilities
func (t *Trivy) ImageLocal(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected (0)
	// +optional
	exitCode int,
	// the type of format to use when generating the compliance report (table)
	// +optional
	format string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the path to an exported image tar
	// +required
	ref *File,
	// the types of scanner to execute (vuln,secret)
	// +optional
	scanners string,
	// the severity of security issues to detect (UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL)
	// +optional
	severity string,
	// a custom go template to use when generating the compliance report
	// +optional
	template string,
	// the types of vulnerabilities to scan for (os,library)
	// +optional
	vulnType string,
) (string, error) {
	cmd := []string{"image", "--input", "image.tar"}

	sargs := scanArgs{
		ExitCode:      exitCode,
		Format:        format,
		IgnoreFile:    t.IgnoreFile,
		IgnoreUnfixed: ignoreUnfixed,
		Scanners:      scanners,
		Severity:      severity,
		Template:      template,
		VulnType:      vulnType,
	}
	cmd = append(cmd, sargs.Args()...)

	return t.Base.
		WithMountedFile("image.tar", ref).
		WithExec(cmd).
		Stdout(ctx)
}

// Scan a filesystem for any vulnerabilities
func (t *Trivy) Filesystem(
	ctx context.Context,
	// the path to directory to scan
	// +required
	dir *Directory,
	// the returned exit code when vulnerabilities are detected (0)
	// +optional
	exitCode int,
	// the type of format to use when generating the compliance report (table)
	// +optional
	format string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the types of scanner to execute (vuln,secret)
	// +optional
	scanners string,
	// the severity of security issues to detect (UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL)
	// +optional
	severity string,
	// a custom go template to use when generating the compliance report
	// +optional
	template string,
	// the types of vulnerabilities to scan for (os,library)
	// +optional
	vulnType string) (string, error) {
	cmd := []string{"filesystem", "."}

	sargs := scanArgs{
		ExitCode:      exitCode,
		Format:        format,
		IgnoreFile:    t.IgnoreFile,
		IgnoreUnfixed: ignoreUnfixed,
		Scanners:      scanners,
		Severity:      severity,
		Template:      template,
		VulnType:      vulnType,
	}
	cmd = append(cmd, sargs.Args()...)

	return t.Base.
		WithDirectory(TrivyWorkDir, dir).
		WithExec(cmd).
		Stdout(ctx)
}
