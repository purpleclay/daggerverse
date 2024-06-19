// A swiss army knife of functions for working with Rust projects.

package main

import (
	"context"
	"fmt"
)

const (
	rustWorkDir = "/src"

	CargoRegistryCache = "/root/.cargo/registry"
	CargoGitCache      = "/root/.cargo/git"
	RustGithubRepo     = "rust-lang/rust"
	RustBaseImage      = "rust"
)

// Rust dagger module
type Rust struct {
	// a custom base image containing an installation of rust
	// +private
	Base *Container

	// a path to a directory containing the projects source code
	// +private
	Src *Directory
}

// Initializes the rust dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of rust. If no image is provided
	// the `rust:<LATEST_TAG>-alpine3.20` will be used. The default image will use musl
	// to support static compilation of Rust binaries. It comes bundled with the following
	// packages: `cmake`, `build-base`, `libressl-dev`, `musl-dev`, and `pkgconfig`
	// +optional
	base *Container,
	// a path to a directory containing the projects source code
	// +required
	src *Directory,
) (*Rust, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		_, err = base.WithoutEntrypoint().WithExec([]string{"rustc", "--version"}).Sync(ctx)
	}

	if err != nil {
		return nil, err
	}

	base = base.WithUser("root").
		WithoutEnvVariable("CARGO_HOME").
		WithDirectory(rustWorkDir, src).
		WithWorkdir(rustWorkDir).
		WithoutEntrypoint()

	base = mountCaches(ctx, base)
	return &Rust{Base: base, Src: src}, nil
}

func defaultImage(ctx context.Context) (*Container, error) {
	tag, err := dag.Github().GetLatestRelease(RustGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From(fmt.Sprintf("%s:%s-alpine3.20", RustBaseImage, tag)).
		WithExec([]string{
			"apk",
			"add",
			"--no-cache",
			"cmake",
			"build-base",
			"libressl-dev",
			"musl-dev",
			"pkgconfig",
		}).
		Sync(ctx)
}

func mountCaches(ctx context.Context, base *Container) *Container {
	cargoRegistry := dag.CacheVolume("cargo_registry")
	cargoGit := dag.CacheVolume("cargo_git")

	return base.
		WithMountedCache(CargoRegistryCache, cargoRegistry).
		WithMountedCache(CargoGitCache, cargoGit)
}

// Lint your Rust project with Clippy to detect common mistakes and to improve
// your Rust code
func (r *Rust) Clippy(
	ctx context.Context,
	// run clippy on the current crate only and not against its dependencies
	// +optional
	noDeps bool,
) (string, error) {
	ctr := r.Base
	if _, err := ctr.WithExec([]string{"cargo", "clippy", "--version"}).Sync(ctx); err != nil {
		ctr = ctr.WithExec([]string{"rustup", "component", "add", "clippy"})
	}

	cmd := []string{"cargo", "clippy"}
	if noDeps {
		cmd = append(cmd, "--no-deps")
	}

	return ctr.WithExec(cmd).Stderr(ctx)
}
