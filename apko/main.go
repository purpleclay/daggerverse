// Build and Publish OCI Container Images from apk packages
package main

import (
	"context"
	"dagger/apko/internal/dagger"
	"fmt"
	"strings"

	"chainguard.dev/apko/pkg/build/types"
	"gopkg.in/yaml.v3"
)

// Apko Dagger Module
type Apko struct{}

// Represents an Apko configuration that forms the basis of all apko commands
type ApkoConfig struct {
	// +private
	Cfg *File
}

// Loads a pre-configured apko configuration file
func (a *Apko) Load(
	// the path to the apko configuration file
	// +required
	cfg *File) *ApkoConfig {
	return &ApkoConfig{Cfg: cfg}
}

// Generates and loads a pre-configured apko configuration file for
// building an image based on the Wolfi OS
//
// Examples:
//
// # Generate a default Wolfi OS apko configuration file
// $ dagger call with-wolfi --entrypoint="/bin/sh -l"
//
// # Extend the default Wolfi OS apko configuration file
// $ dagger call with-wolfi --entrypoint="echo \$VAR1" --env="VAR1:VALUE1"
func (a *Apko) WithWolfi(
	// the command to execute after the container entrypoint
	// +optional
	cmd string,
	// the entrypoint to the container
	// +required
	entrypoint string,
	// a list of environment variables to set within the container image, expected in (key:value) format
	// +optional
	env []string,
) (*ApkoConfig, error) {
	environment := map[string]string{}
	if len(env) > 0 {
		for _, e := range env {
			key, value, found := strings.Cut(e, ":")
			if !found {
				return nil, fmt.Errorf("failed to parse malformed environment variable argument", e)
			}
			environment[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}

	cfg := types.ImageConfiguration{
		Contents: types.ImageContents{
			Repositories: []string{"https://packages.wolfi.dev/os"},
			Keyring:      []string{"https://packages.wolfi.dev/os/wolfi-signing.rsa.pub"},
			Packages: []string{
				"wolfi-base",
				"ca-certificates-bundle",
			},
		},
		Entrypoint: types.ImageEntrypoint{
			Command: entrypoint,
		},
		Cmd: cmd,
		Archs: []types.Architecture{
			types.ParseArchitecture("x86_64"),
		},
		Environment: environment,
	}

	out, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, err
	}

	dir := dag.Directory().
		WithNewFile("apko.yaml", string(out), dagger.DirectoryWithNewFileOpts{Permissions: 0o644})

	return &ApkoConfig{
		Cfg: dir.File("apko.yaml"),
	}, nil
}

// Prints the generated apko configuration file to stdout
func (a *ApkoConfig) Yaml(ctx context.Context) (string, error) {
	return a.Cfg.Contents(ctx)
}

// Builds an image from an apko configuration file and outputs it as a file
// that can be imported using:
//
// $ docker load < image.tar
//
// Examples:
//
// # Build an OCI image from a provided apko configuration file
// $ dagger call load --cfg apko.yaml build --ref registry:5000/example:latest
//
// # Build an OCI image based on the Wolfi OS
// $ dagger call with-wolfi build --ref registry:5000/example:latest
func (a *ApkoConfig) Build(
	// additional OCI annotations to add to the built image, expected in (key:value) format
	// +optional
	annotations []string,
	// a list of architectures to build, overwriting the config
	// +optional
	archs []string,
	// a list of additional packages to include within the built image
	// +optional
	pkgs []string,
	// a list of additional repositories used to pull packages into the built image
	// +optional
	repos []string,
	// the image reference to build
	// +required
	ref string,
	// detect and embed VCS URLs within the built OCI image
	// +optional
	// default=true
	vcs bool,
	// generate and embed an SBOM (software bill of materials) within the built OCI image
	// +optional
	// +default=true
	sbom bool,
) *Directory {
	cmd := []string{
		"build",
		"/apko/apko.yaml",
		ref,
		imageFromRef(ref),
	}
	cmd = append(cmd, formatArgs(annotations, archs, pkgs, repos, ref, vcs, sbom)...)

	return dag.Container().
		From("cgr.dev/chainguard/apko").
		WithWorkdir("apko").
		WithFile("apko.yaml", a.Cfg).
		WithExec(cmd).
		Directory("")
}

func imageFromRef(ref string) string {
	image := ref
	if pos := strings.LastIndex(image, "/"); pos > -1 {
		image = image[pos+1:]
	}

	if pos := strings.LastIndex(image, ":"); pos > -1 {
		image = image[:pos]
	}
	return image + ".tar"
}

func formatArgs(annotations, archs, pkgs, repos []string, ref string, vcs, sbom bool) []string {
	var args []string

	if len(archs) > 0 {
		args = append(args, "--arch", strings.Join(archs, ","))
	}

	if len(repos) > 0 {
		args = append(args, "--repository-append", strings.Join(repos, ","))
	}

	if len(pkgs) > 0 {
		args = append(args, "--package-append", strings.Join(pkgs, ","))
	}

	if len(annotations) > 0 {
		args = append(args, "--annotations", strings.Join(annotations, ","))
	}

	if !sbom {
		args = append(args, "--sbom=false")
	}

	if !vcs {
		args = append(args, "--vcs=false")
	}

	return args
}

// Builds an image from an apko configuration file and publishes it to an OCI
// image registry
//
// Examples:
//
// # Publish an OCI image from a provided apko configuration file
// $ dagger call load --cfg apko.yaml publish --ref registry:5000/example:latest
//
// # Publish an OCI image based on the Wolfi OS
// $ dagger call with-wolfi build --ref registry:5000/example:latest
func (a *ApkoConfig) Publish(
	ctx context.Context,
	// additional OCI annotations to add to the built image, expected in (key:value) format
	// +optional
	annotations []string,
	// a list of architectures to build, overwriting the config
	// +optional
	archs []string,
	// a list of additional packages to include within the built image
	// +optional
	pkgs []string,
	// a list of additional repositories used to pull packages into the built image
	// +optional
	repos []string,
	// the image reference to build
	// +required
	ref string,
	// detect and embed VCS URLs within the built OCI image
	// +optional
	// default=true
	vcs bool,
	// generate and embed an SBOM (software bill of materials) within the built OCI image
	// +optional
	// +default=true
	sbom bool,
	// the address of the registry to authenticate with
	// +optional
	// +default="docker.io"
	registry,
	// the username for authenticating with the registry
	// +optional
	username string,
	// the password for authenticating with the registry
	// +optional
	password *dagger.Secret,
) (string, error) {
	cmd := []string{
		"publish",
		"/apko/apko.yaml",
		ref,
	}
	cmd = append(cmd, formatArgs(annotations, archs, pkgs, repos, ref, vcs, sbom)...)

	ctr := dag.Container().
		From("cgr.dev/chainguard/apko")

	if registry != "" && username != "" && password != nil {
		ctr = ctr.WithRegistryAuth(registry, username, password)
	}

	return ctr.
		WithWorkdir("apko").
		WithFile("apko.yaml", a.Cfg).
		WithExec(cmd).
		Stdout(ctx)
}
