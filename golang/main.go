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
	"fmt"
	"runtime"
	"strings"
)

const (
	// Prior to go 1.21, the go.mod doesn't include the full version, so build
	// against the latest possible version
	go1_17 = "1.17.13-bullseye"
	go1_18 = "1.18.10-bullseye"
	go1_19 = "1.19.13-bullseye"
	go1_20 = "1.20.13-bookworm"
)

// Golang dagger function
type Golang struct {
	// Base is the image used by all golang dagger functions, defaults to the bookworm base image
	// +private
	Base *Container

	// Src is a directory that contains the projects source code
	// +private
	Src *Directory
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
		version, _ := g.ModVersion(context.Background())
		g.Base = base(version)
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
