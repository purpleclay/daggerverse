// A swiss army knife of functions for working with Golang projects.
//
// A collection of functions for building, formatting, testing, linting and scanning
// your Go project for vulnerabilities.
package main

import (
	"context"
	"dagger/golang/internal/dagger"
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

	goMod     = "go.mod"
	goWorkDir = "/src"
)

// Golang dagger module
type Golang struct {
	// Base is the image used by all golang dagger functions, defaults to the bookworm base image
	// +private
	Base *dagger.Container

	// Src is a directory that contains the projects source code
	// +private
	Src *dagger.Directory

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
	//
	// `NOTE:` Any pre-existing entrypoint will be removed, in favour of raw `go` commands
	// +optional
	base *dagger.Container,
	// a path to a directory containing the source code
	// +required
	src *dagger.Directory,
) (*Golang, error) {
	version, err := inspectModVersion(context.Background(), src)
	if err != nil {
		return nil, err
	}

	if base == nil {
		base = defaultImage(version)
	} else {
		if _, err = base.WithoutEntrypoint().WithExec([]string{"go", "version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	// Ensure cache mounts are configured for any type of image
	base = mountCaches(ctx, base).
		WithDirectory(goWorkDir, src).
		WithWorkdir(goWorkDir).
		WithoutEntrypoint()

	return &Golang{Base: base, Src: src, Version: version}, nil
}

func inspectModVersion(ctx context.Context, src *dagger.Directory) (string, error) {
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

func mountCaches(ctx context.Context, base *dagger.Container) *dagger.Container {
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
//
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . mod-version`
func (g *Golang) ModVersion() string {
	return g.Version
}

func defaultImage(version string) *dagger.Container {
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

// Build a static binary from a Go project using the provided configuration.
// A directory is returned containing the build binary.
//
// Build a binary from a main.go file located at the project root:
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . build`
//
// Build a binary targeting a custom os and architecture:
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . build --os linux --arch arm64`
//
// Build a binary from a main.go file located within a cmd folder:
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . build --main cmd/example/main.go --out example`
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
	arch string,
	// flags to configure the linking during a build, by default sets flags for
	// generating a release binary
	// +optional
	// +default=["-s", "-w"]
	ldflags []string,
) *dagger.Directory {
	if os == "" {
		os = runtime.GOOS
	}

	if arch == "" {
		arch = runtime.GOARCH
	}

	cmd := []string{"go", "build", "-ldflags", strings.Join(ldflags, " ")}
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
		WithExec(cmd).
		Directory(goWorkDir)
}

// Execute tests defined within the target project, ignores benchmarks by default
//
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . test`
//
// Execute only short running tests ensuring they are shuffled:
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . test --short --shuffle`
//
// Execute a single test:
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . test --run 'TestSingleFeature'
func (g *Golang) Test(
	ctx context.Context,
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
) (string, error) {
	cmd := []string{"go", "test", "-vet=off", "-covermode=atomic", "./..."}
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

	return g.Base.WithExec(cmd).Stdout(ctx)
}

// Execute benchmarks defined within the target project, excludes all other tests
//
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . bench`
func (g *Golang) Bench(
	ctx context.Context,
	// print memory allocation statistics for benchmarks
	// +optional
	// +default=true
	memory bool,
	// the time.Duration each benchmark should run for
	// +optional
	// +default="5s"
	time string,
) (string, error) {
	cmd := []string{"go", "test", "-bench=.", "-benchtime", time, "-run=^#", "./..."}
	if memory {
		cmd = append(cmd, "-benchmem")
	}

	return g.Base.WithExec(cmd).Stdout(ctx)
}

// Scans the target project for vulnerabilities using govulncheck
//
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . vulncheck`
func (g *Golang) Vulncheck(ctx context.Context) (string, error) {
	if g.Version == "1.17" {
		return "", fmt.Errorf("govulncheck supports go versions 1.18 and higher")
	}

	ctr := g.Base
	if _, err := ctr.WithExec([]string{"govulncheck", "--version"}).Sync(ctx); err != nil {
		tag, err := dag.Github().GetLatestRelease("golang/vuln").Tag(ctx)
		if err != nil {
			return "", err
		}

		ctr = ctr.WithExec([]string{"go", "install", "golang.org/x/vuln/cmd/govulncheck@" + tag})
	}

	return ctr.
		WithExec([]string{"govulncheck", "./..."}).
		Stdout(ctx)
}

// Lint the target project using golangci-lint
//
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . lint`
func (g *Golang) Lint(
	ctx context.Context,
	// the type of report that should be generated
	// +optional
	// +default="colored-line-number"
	format string,
) (string, error) {
	ctr := g.Base
	if _, err := ctr.WithExec([]string{"golangci-lint", "version"}).Sync(ctx); err != nil {
		tag, err := dag.Github().GetLatestRelease("golangci/golangci-lint").Tag(ctx)
		if err != nil {
			return "", err
		}

		// Install using the recommended approach: https://golangci-lint.run/welcome/install/
		cmd := []string{
			"curl",
			"-sSfL",
			"https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh",
			"|",
			"sh",
			"-s",
			"--",
			"-b",
			"$(go env GOPATH)/bin",
			tag,
		}
		ctr = ctr.WithExec([]string{"bash", "-c", strings.Join(cmd, " ")})
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
	}

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Format the source code within a target project using gofumpt. Formatted code must be
// copied back onto the host.
//
// `dagger call -m github.com/purpleclay/daggerverse/golang --src . format export --path .`
func (g *Golang) Format(ctx context.Context) (*dagger.Directory, error) {
	ctr := g.Base
	if _, err := ctr.WithExec([]string{"gofumpt", "-version"}).Sync(ctx); err != nil {
		tag, err := dag.Github().GetLatestRelease("mvdan/gofumpt").Tag(ctx)
		if err != nil {
			return nil, err
		}

		ctr = ctr.WithExec([]string{"go", "install", "mvdan.cc/gofumpt@" + tag})
	}

	cmd := []string{"gofumpt", "-w", "-d", "."}

	return ctr.WithExec(cmd).Directory(goWorkDir), nil
}
