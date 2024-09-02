// Build and Publish OCI Container Images from apk packages
package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/apko/internal/dagger"

	"chainguard.dev/apko/pkg/build/types"
	"gopkg.in/yaml.v3"
)

// Apko Dagger Module
type Apko struct{}

// Represents an Apko configuration that forms the basis of all apko commands
type ApkoConfig struct {
	// +private
	Cfg *dagger.File
}

// Loads a pre-configured apko configuration file
func (a *Apko) Load(
	// the path to the apko configuration file
	// +required
	cfg *dagger.File,
) *ApkoConfig {
	return &ApkoConfig{Cfg: cfg}
}

type imageConfig struct {
	Archs        []string
	Repositories []string
	Keyring      []string
	Packages     []string
	Entrypoint   string
	Cmd          string
	Env          []string
}

// Generates and loads a pre-configured apko configuration file for
// building an image based on the Wolfi OS.  By default, the
// [wolfi-base, ca-certificates-bundle] packages will be installed.
//
// Examples:
//
// # Generate a default Wolfi OS apko configuration file
// $ dagger call with-wolfi --entrypoint="/bin/sh -l"
//
// # Extend the default Wolfi OS apko configuration file
// $ dagger call with-wolfi --entrypoint="echo \$VAR1" --env="VAR1:VALUE1"
func (a *Apko) WithWolfi(
	// a list of container architectures (defaults to amd64)
	// +optional
	archs []string,
	// the command to execute after the container entrypoint
	// +optional
	cmd string,
	// the entrypoint to the container
	// +required
	entrypoint string,
	// a list of environment variables to set within the container image, expected in (key:value) format
	// +optional
	env []string,
	// a list of packages to install within the container
	// +optional
	pkgs []string,
) (*ApkoConfig, error) {
	packages := append([]string{
		"wolfi-base",
		"ca-certificates-bundle",
	}, pkgs...)

	wolfi := imageConfig{
		Archs:        archs,
		Repositories: []string{"https://packages.wolfi.dev/os"},
		Keyring:      []string{"https://packages.wolfi.dev/os/wolfi-signing.rsa.pub"},
		Packages:     packages,
		Entrypoint:   entrypoint,
		Cmd:          cmd,
		Env:          env,
	}

	cfg, err := toFile(wolfi)
	if err != nil {
		return nil, err
	}

	return &ApkoConfig{Cfg: cfg}, nil
}

func toFile(cfg imageConfig) (*dagger.File, error) {
	environment := map[string]string{
		"PATH": "/usr/sbin:/sbin:/usr/local/bin:/usr/bin:/bin",
	}
	if len(cfg.Env) > 0 {
		for _, e := range cfg.Env {
			key, value, found := strings.Cut(e, ":")
			if !found {
				return nil, fmt.Errorf("failed to parse malformed environment variable argument", e)
			}
			environment[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}

	if len(cfg.Archs) == 0 {
		cfg.Archs = append(cfg.Archs, "amd64")
	}

	var archs []types.Architecture
	for _, arch := range cfg.Archs {
		archs = append(archs, types.ParseArchitecture(arch))
	}

	imgCfg := types.ImageConfiguration{
		Contents: types.ImageContents{
			Repositories: cfg.Repositories,
			Keyring:      cfg.Keyring,
			Packages:     cfg.Packages,
		},
		Entrypoint: types.ImageEntrypoint{
			Command: cfg.Entrypoint,
		},
		Cmd:         cfg.Cmd,
		Archs:       archs,
		Environment: environment,
	}

	out, err := yaml.Marshal(&imgCfg)
	if err != nil {
		return nil, err
	}

	return dag.Directory().
		WithNewFile("apko.yaml", string(out), dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("apko.yaml"), nil
}

// Generates and loads a pre-configured apko configuration file for
// building an image based on the Alpine OS. By default, the
// [alpine-base, ca-certificates-bundle] packages will be installed.
//
// Examples:
//
// # Generate a default alpine OS apko configuration file
// $ dagger call with-alpine --entrypoint="/bin/sh -l"
//
// # Extend the default alpine OS apko configuration file
// $ dagger call with-alpine --entrypoint="echo \$VAR1" --env="VAR1:VALUE1"
func (a *Apko) WithAlpine(
	// a list of container architectures (defaults to amd64)
	// +optional
	archs []string,
	// the command to execute after the container entrypoint
	// +optional
	cmd string,
	// the entrypoint to the container
	// +required
	entrypoint string,
	// a list of environment variables to set within the container image, expected in (key:value) format
	// +optional
	env []string,
	// a list of packages to install within the container
	// +optional
	pkgs []string,
) (*ApkoConfig, error) {
	packages := append([]string{
		"alpine-base",
		"ca-certificates-bundle",
	}, pkgs...)

	alpine := imageConfig{
		Archs:        archs,
		Repositories: []string{"https://dl-cdn.alpinelinux.org/alpine/edge/main"},
		Packages:     packages,
		Entrypoint:   entrypoint,
		Cmd:          cmd,
		Env:          env,
	}

	cfg, err := toFile(alpine)
	if err != nil {
		return nil, err
	}

	return &ApkoConfig{Cfg: cfg}, nil
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
) *dagger.Directory {
	cmd := []string{
		"apko",
		"build",
		"/apko/apko.yaml",
		ref,
		imageFromRef(ref),
	}
	cmd = append(cmd, formatArgs(annotations, archs, pkgs, repos, ref, vcs, sbom)...)

	return base().
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

func base() *dagger.Container {
	return dag.Container().
		From("cgr.dev/chainguard/wolfi-base").
		WithExec([]string{"apk", "add", "--no-cache", "apko"}).
		WithWorkdir("apko")
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
		"apko",
		"publish",
		"/apko/apko.yaml",
		ref,
	}
	cmd = append(cmd, formatArgs(annotations, archs, pkgs, repos, ref, vcs, sbom)...)

	ctr := base()

	if registry != "" && username != "" && password != nil {
		ctr = ctr.WithEnvVariable("REGISTRY", registry).
			WithEnvVariable("REGISTRY_USER", username).
			WithSecretVariable("REGISTRY_PASSWORD", password).
			WithExec([]string{"sh", "-c", "apko login $REGISTRY -u $REGISTRY_USER -p $REGISTRY_PASSWORD"})
	}

	return ctr.
		WithFile("apko.yaml", a.Cfg).
		WithExec(cmd).
		Stdout(ctx)
}
