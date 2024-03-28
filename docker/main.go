// Manage your docker based projects
//
// A collection of functions for building, saving and publishing your Docker based projects
package main

import (
	"context"
	"dagger/docker/internal/dagger"
	"strings"
)

// Docker dagger module
type Docker struct{}

// DockerBuild contains an image built from the provided Dockerfile,
// it serves as an intermediate type for chaining other functions
type DockerBuild struct {
	// +private
	Image *Container
}

// Build an image using a Dockerfile. Supports cross-compilation
func (d *Docker) Build(
	// the path to a directory that will be used as the docker context
	// +required
	src *Directory,
	// the path to the Dockfile
	// +default="Dockerfile"
	// +required
	file string,
	// a list of build arguments in the format of arg=value
	// +optional
	args []string,
	// the name of a target build stage
	// +optional
	target string,
	// the target platform
	// +optional
	// +default="linux/amd64"
	platform string) *DockerBuild {
	var buildArgs []dagger.BuildArg
	if len(args) > 0 {
		for _, arg := range args {
			if name, value, found := strings.Cut(arg, "="); found {
				buildArgs = append(buildArgs, dagger.BuildArg{
					Name:  strings.TrimSpace(name),
					Value: strings.TrimSpace(value),
				})
			}
		}
	}

	con := dag.Container(dagger.ContainerOpts{Platform: dagger.Platform(platform)}).
		Build(src, dagger.ContainerBuildOpts{
			BuildArgs:  buildArgs,
			Dockerfile: file,
			Target:     target,
		})

	return &DockerBuild{Image: con}
}

// Retrieves the underlying container built from a Dockerfile
func (m *DockerBuild) Base() *Container {
	return m.Image
}

// Save the built image as a tarball ready for exporting
func (m *DockerBuild) Save() *File {
	return m.Image.AsTarball()
}

// Publish the built image to a target registry
func (m *DockerBuild) Publish(
	ctx context.Context,
	// the image reference to publish
	// +required
	ref string) (string, error) {
	return m.Image.Publish(ctx, ref)
}
