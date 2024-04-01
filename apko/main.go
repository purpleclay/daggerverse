// Build and Publish OCI Container Images from apk packages
package main

import (
	"dagger/apko/internal/dagger"
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
func (a *Apko) WithWolfi() (*ApkoConfig, error) {
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
			Command: "/bin/sh -l",
		},
		Archs: []types.Architecture{
			types.ParseArchitecture("x86_64"),
		},
	}

	out, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, err
	}

	// TODO: change Cfg to be a *Directory
	dir := dag.Directory().
		WithNewFile("apko.yaml", string(out), dagger.DirectoryWithNewFileOpts{Permissions: 0o644})

	return &ApkoConfig{
		Cfg: dir.File("apko.yaml"),
	}, nil
}

// Builds an image from an apko configuration file and outputs it as a file
// that can be imported using:
//
// $ docker load < image.tar
//
// Examples:
//
// dagger call load --cfg apko.yaml build --ref registry:5000/example:latest
func (a *ApkoConfig) Build(
	// the image reference to build
	// +required
	ref string) *File {
	image := ref
	if pos := strings.LastIndex(image, "/"); pos > -1 {
		image = image[pos+1:]
	}

	if pos := strings.LastIndex(image, ":"); pos > -1 {
		image = image[:pos]
	}
	image = image + ".tar"

	return dag.Container().
		WithMountedFile("apko.yaml", a.Cfg).
		From("cgr.dev/chainguard/apko").
		WithExec([]string{"build", "apko.yaml", ref, image}).
		File(image)
}
