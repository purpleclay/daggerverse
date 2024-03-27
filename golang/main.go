// A collection of functions for building, testing, linting and scanning your Go project
// for vulnerabilities.
//
// To select a Go project, a path must be provided for the --src flag. The base image used
// by all functions is automatically resolved from the project version defined within the go.mod
// file. Auto-detection is supported for Go 1.17 and above:
//
// - >= 1.17 < 1.20: the Debian bullseye image is used.
// - >= 1.20: the Debian bookworm image is used.
//
// Set the --image flag to switch to using your custom image.
package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

const (
	// Prior to go 1.21, the go.mod doesn't include the full version, so build
	// against the latest possible version
	go1_17 = "golang:1.17.13-bullseye"
	go1_18 = "golang:1.18.10-bullseye"
	go1_19 = "golang:1.19.13-bullseye"
	go1_20 = "golang:1.20.13-bookworm"
)

// Golang dagger module
type Golang struct {
	// Base is the image used by all golang dagger functions, defaults to the bookworm base image
	// +private
	Base *Container

	// Src is a directory that contains the projects source code
	// +private
	Src *Directory

	// Version of the go project
	// +private
	Version string
}

// New initializes the golang dagger module
func New(
	// a custom base image containing an installation of golang
	// +optional
	image *Container,
	// a path to a directory containing the source code
	// +required
	src *Directory) *Golang {
	g := &Golang{Base: image, Src: src}
	if g.Base == nil {
		// Detect the version of Go and select the right base image
		g.Version, _ = g.ModVersion(context.Background())
		g.Base = base(g.Version)
	}

	return g
}

// Echoes the version of go used by a target project
func (g *Golang) ModVersion(ctx context.Context) (string, error) {
	return dag.Container().
		From("busybox").
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"ash", "-c", "grep -E '^go' go.mod | awk '{printf $2}'"}).
		Stdout(ctx)
}

func base(version string) *Container {
	// These are mapped directly to GOCACHE and GOMOD environment variables
	mod := dag.CacheVolume("gomod")
	build := dag.CacheVolume("gobuild")

	var image string
	switch version {
	case "1.17":
		image = go1_17
	case "1.18":
		image = go1_18
	case "1.19":
		image = go1_19
	case "1.20":
		image = go1_20
	default:
		image = fmt.Sprintf("golang:%s-bookworm", version)
	}

	return dag.Container().
		From(image).
		WithMountedCache("/go/pkg/mod", mod).
		WithMountedCache("/root/.cache/go-build", build)
}

// Build a static release binary without debug information or symbols
func (g *Golang) Build(
	// the path to the main.go file of the project
	// +optional
	main string,
	// the name of the built binary
	// +optional
	out string,
	// the target operating system
	// +optional
	os string,
	// the target architecture
	// +optional
	arch string) *Directory {
	if os == "" {
		os = runtime.GOOS
	}

	if arch == "" {
		arch = runtime.GOARCH
	}

	cmd := []string{"go", "build", "-ldflags", "-s -w"}
	if out != "" {
		cmd = append(cmd, "-o", out)
	}

	if main != "" {
		cmd = append(cmd, main)
	}

	return g.Base.
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", os).
		WithEnvVariable("GOARCH", arch).
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Directory("/src")
}

// Execute tests defined within the target project, ignores benchmarks by default
func (g *Golang) Test(
	// if only short running tests should be executed
	// +optional
	// +default=true
	short bool,
	// if the tests should be executed out of order
	// +optional
	// +default=true
	shuffle bool,
	// run select tests only, defined using a regex
	// +optional
	run string,
	// skip select tests, defined using a regex
	// +optional
	skip string,
	// log all output from tests even if there are successful
	// +optional
	verbose bool,
) *Directory {
	cmd := []string{"go", "test", "-vet=off", "-covermode=atomic", "-coverprofile=coverage.out", "-json", "./..."}
	if short {
		cmd = append(cmd, "-short")
	}

	if shuffle {
		cmd = append(cmd, "-shuffle=on")
	}

	if run != "" {
		cmd = append(cmd, []string{"-run", run}...)
	}

	if skip != "" {
		cmd = append(cmd, []string{"-skip", skip}...)
	}

	if verbose {
		cmd = append(cmd, "-v")
	}

	// Capture JSON report and pipe it into tparse
	cmd = append(cmd, []string{"|", "tee", "test-report.json", "|", "tparse", "-follow"}...)

	return g.Base.
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"go", "install", "github.com/mfridman/tparse@latest"}).
		WithExec([]string{"go", "install", "gotest.tools/gotestsum@latest"}).
		WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).
		WithExec([]string{"gotestsum", "--junitfile", "junit-report.xml", "--raw-command", "cat", "test-report.json"}).
		Directory("/src")
}

// Execute benchmarks defined within the target project, excludes all other tests
func (g *Golang) Bench(
	// print memory allocation statistics for benchmarks
	// +optional
	// +default=true
	memory bool,
	// the time.Duration each benchmark should run for
	// +optional
	// +default="5s"
	time string) *Directory {
	cmd := []string{"go", "test", "-bench=.", "-benchtime", time, "-run=^#", "./..."}
	if memory {
		cmd = append(cmd, "-benchmem")
	}

	cmd = append(cmd, []string{"|", "tee", "bench.out"}...)

	return g.Base.
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).
		Directory("/src")
}

// Scans the target project for vulnerabilities using govulncheck
func (g *Golang) Vulncheck() *Directory {
	cmd := []string{"govulncheck", "./...", "|", "tee", "vulncheck.out"}

	return g.Base.
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"}).
		WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).
		Directory("/src")
}

// Lint the target project using golangci-lint
func (g *Golang) Lint(ctx context.Context) *File {
	// Install using the recommended approach: https://golangci-lint.run/welcome/install/
	installCmd := []string{
		"curl",
		"-sSfL",
		"https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh",
		"|",
		"sh",
		"-s",
		"--",
		"-b",
		"$(go env GOPATH)/bin",
	}

	cmd := []string{"golangci-lint", "run", "--timeout", "5m", "--go", g.Version, "|", "tee", "lint.out"}

	return g.Base.
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"bash", "-c", strings.Join(installCmd, " ")}).
		WithExec([]string{"bash", "-o", "pipefail", "-c", strings.Join(cmd, " ")}).
		File("lint.out")
}
