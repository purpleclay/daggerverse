/*
Copyright (c) 2024 Purple Clay

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"context"
	"dagger/trivy/internal/dagger"
	"strconv"
)

// Trivy dagger module
type Trivy struct {
	// Base is the image used by all trivy dagger functions
	// +private
	Base *Container
}

// New initializes the golang dagger module
func New(
	// a custom base image containing an installation of trivy
	// +optional
	image *Container) *Trivy {
	g := &Trivy{Base: image}
	if g.Base == nil {
		g.Base = base()
	}

	return g
}

func base() *Container {
	pkgs := []string{"ca-certificates", "git", "trivy"}

	return dag.Wolfi().
		Container(dagger.WolfiContainerOpts{Packages: pkgs}).
		WithMountedCache("/root/.cache/trivy", dag.CacheVolume("trivydb"))
}

// Scan a published (or remote) image for any vulnerabilities
func (t *Trivy) Image(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected
	// +optional
	// +default=1
	exitCode int,
	// the reference to an image within a repository
	// +required
	ref string,
	// the types of scanner to execute
	// +optional
	// +default="vuln"
	scanners string,
	// the severity of security issues to detect
	// +optional
	// +default="HIGH,CRITICAL"
	severity string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the types of vulnerabilities to scan for
	// +optional
	// +default="os,library"
	vulnType string) (string, error) {
	cmd := []string{
		"trivy",
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
	}
	if ignoreUnfixed {
		cmd = append(cmd, "--ignore-unfixed")
	}

	return t.Base.WithExec(cmd).Stdout(ctx)
}

// Scan a locally exported image for any vulnerabilities
func (t *Trivy) ImageLocal(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected
	// +optional
	// +default=1
	exitCode int,
	// the path to an exported image tar
	// +required
	ref *File,
	// the types of scanner to execute
	// +optional
	// +default="vuln"
	scanners string,
	// the severity of security issues to detect
	// +optional
	// +default="HIGH,CRITICAL"
	severity string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the types of vulnerabilities to scan for
	// +optional
	// +default="os,library"
	vulnType string) (string, error) {
	cmd := []string{
		"trivy",
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
	}
	if ignoreUnfixed {
		cmd = append(cmd, "--ignore-unfixed")
	}

	return t.Base.
		WithMountedFile("/scan/image.tar", ref).
		WithExec(cmd).
		Stdout(ctx)
}

// Scan a filesystem for any vulnerabilities
func (t *Trivy) Filesystem(
	ctx context.Context,
	// the returned exit code when vulnerabilities are detected
	// +optional
	// +default=1
	exitCode int,
	// the path to directory
	// +required
	ref *Directory,
	// the types of scanner to execute
	// +optional
	// +default="vuln"
	scanners string,
	// the severity of security issues to detect
	// +optional
	// +default="HIGH,CRITICAL"
	severity string,
	// filter out any vulnerabilities without a known fix
	// +optional
	ignoreUnfixed bool,
	// the types of vulnerabilities to scan for
	// +optional
	// +default="os,library"
	vulnType string) (string, error) {
	cmd := []string{
		"trivy",
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
	}
	if ignoreUnfixed {
		cmd = append(cmd, "--ignore-unfixed")
	}

	return t.Base.
		WithDirectory("/scan", ref).
		WithWorkdir("/scan").
		WithExec(cmd).
		Stdout(ctx)
}
