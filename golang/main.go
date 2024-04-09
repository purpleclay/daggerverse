// Build, test, lint and scan your Go project for vulnerabilities.
//
// A collection of functions for building, testing, linting and scanning your Go project for
// vulnerabilities.
package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/mod/modfile"
)

const (
	// Prior to go 1.21, the go.mod doesn't include the full version, so build
	// against the latest possible version
	go1_17 = "golang:1.17.13-bullseye"
	go1_18 = "golang:1.18.10-bullseye"
	go1_19 = "golang:1.19.13-bullseye"
	go1_20 = "golang:1.20.13-bookworm"

	goMod = "go.mod"
)

// Golang dagger module
type Golang struct {
	// Base is the image used by all golang dagger functions, defaults to the bookworm base image
	// +private
	Base *Container

	// Src is a directory that contains the projects source code
	// +private
	Src *Directory

	// Version of the go project, defined within the go.mod file
	// +private
	Version string
}

// New initializes the golang dagger module
func New(
	ctx context.Context,
	// A custom base image containing an installation of golang. If no image is provided,
	// one is resolved based on the Go version defined within the projects go.mod file. The
	// official Go image is pulled from DockerHub using either the bullseye (< 1.20) or
	// bookworm (> 1.20) variants.
	// +optional
	base *Container,
	// a path to a directory containing the source code
	// +required
	src *Directory) (*Golang, error) {

	version, err := inspectModVersion(context.Background(), src)
	if err != nil {
		return nil, err
	}

	if base == nil {
		base = defaultImage(version)
	} else {
		if _, err = base.WithExec([]string{"go", "version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	// Ensure cache mounts are configured for any type of image
	base = mountCaches(ctx, base)

	return &Golang{Base: base, Src: src, Version: version}, nil
}

func inspectModVersion(ctx context.Context, src *Directory) (string, error) {
	mod, err := src.File(goMod).Contents(ctx)
	if err != nil {
		return "", err
	}

	f, err := modfile.Parse(goMod, []byte(mod), nil)
	if err != nil {
		return "", err
	}
	return f.Go.Version, nil
}

func mountCaches(ctx context.Context, base *Container) *Container {
	goCacheEnv, _ := base.WithExec([]string{"go", "env", "GOCACHE"}).Stdout(ctx)
	goModCacheEnv, _ := base.WithExec([]string{"go", "env", "GOMODCACHE"}).Stdout(ctx)

	gomod := dag.CacheVolume("gomod")
	gobuild := dag.CacheVolume("gobuild")

	return base.
		WithMountedCache(goModCacheEnv, gomod).
		WithMountedCache(goCacheEnv, gobuild)
}

// Echoes the version of go defined within a projects go.mod file.
// It expects the go.mod file to be located within the root of the project
func (g *Golang) ModVersion() string {
	return g.Version
}

func defaultImage(version string) *Container {
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

	return dag.Container().From(image)
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
func (g *Golang) Vulncheck() *File {
	cmd := []string{"govulncheck", "./...", "|", "tee", "vulncheck.out"}

	return g.Base.
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@latest"}).
		WithExec([]string{"sh", "-c", strings.Join(cmd, " ")}).
		File("vulncheck.out")
}

// Lint the target project using golangci-lint
func (g *Golang) Lint(
	ctx context.Context,
	// the type of report that should be generated
	// +optional
	// +default="colored-line-number"
	format string) *File {
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

	cmd := []string{
		"golangci-lint",
		"run",
		"--timeout",
		"5m",
		"--go",
		g.Version,
		"--out-format",
		format,
		"|",
		"tee",
		"lint.out",
	}

	return g.Base.
		WithDirectory("/src", g.Src).
		WithWorkdir("/src").
		WithExec([]string{"bash", "-c", strings.Join(installCmd, " ")}).
		WithExec([]string{"bash", "-o", "pipefail", "-c", strings.Join(cmd, " ")}).
		File("lint.out")
}
