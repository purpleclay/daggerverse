// Manage your docker based projects
//
// A collection of functions for building, saving and publishing your Docker based projects
package main

import (
	"context"
	"fmt"
	"strings"

	"dagger/docker/internal/dagger"
)

// Docker dagger module
type Docker struct {
	// +private
	// +optional
	Auth *DockerAuth
}

// New initializes the docker dagger module. Two options are available
// if authenticating to a private registry. An explicit `docker login`
// can be actioned before invoking this module, or dagger can authenticate
// to the registry if registry authentication details are provided
func New(
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
) *Docker {
	var auth *DockerAuth
	if registry != "" && username != "" && password != nil {
		auth = &DockerAuth{
			Registry: registry,
			Username: username,
			Password: password,
		}
	}

	return &Docker{Auth: auth}
}

// DockerAuth contains credentials for authenticating with a docker registry
type DockerAuth struct {
	// +private
	// +optional
	Registry string
	// +private
	// +optional
	Username string
	// +private
	// +optional
	Password *dagger.Secret
}

// DockerBuild contains an image built from the provided Dockerfile,
// it serves as an intermediate type for chaining other functions. If
// multiple platforms were provided, then multiple images will exist
type DockerBuild struct {
	// +private
	// +required
	Builds []*dagger.Container
	// +private
	// +optional
	Auth *DockerAuth
}

// Build an image using a Dockerfile. Supports multi-platform images
func (d *Docker) Build(
	// the path to a directory that will be used as the docker context
	// +required
	dir *dagger.Directory,
	// the path to the Dockfile
	// +default="Dockerfile"
	// +optional
	file string,
	// a list of build arguments in the format of arg=value
	// +optional
	args []string,
	// the name of a target build stage
	// +optional
	target string,
	// a list of target platforms for cross-compilation
	// +optional
	// +default=["linux/amd64"]
	platform []dagger.Platform,
) *DockerBuild {
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

	var builds []*dagger.Container
	for _, pform := range platform {
		ctr := dag.Container(dagger.ContainerOpts{Platform: pform})
		if d.Auth != nil {
			ctr = ctr.WithRegistryAuth(d.Auth.Registry, d.Auth.Username, d.Auth.Password)
		}

		ctr = ctr.Build(dir, dagger.ContainerBuildOpts{
			BuildArgs:  buildArgs,
			Dockerfile: file,
			Target:     target,
		})

		builds = append(builds, ctr)
	}

	return &DockerBuild{Builds: builds, Auth: d.Auth}
}

// Save the built image as a tarball ready for exporting. A tarball will be generated using
// the following convention `<name>@<platform>.tar` (e.g. image~linux-amd64.tar)
func (d *DockerBuild) Save(
	ctx context.Context,
	// a name for the exported tarball
	// +optional
	// +default="image"
	name string,
) *dagger.Directory {
	imgName := strings.ReplaceAll(name, " ", "-")

	dir := dag.Directory()
	for _, build := range d.Builds {
		platform, _ := build.Platform(ctx)

		dir = dir.WithFile(fmt.Sprintf("%s@%s.tar", imgName, strings.Replace(string(platform), "/", "-", 1)),
			build.AsTarball(dagger.ContainerAsTarballOpts{
				ForcedCompression: dagger.Gzip,
			}),
			dagger.DirectoryWithFileOpts{Permissions: 0o644},
		)
	}

	return dir
}

// Retrieves a built image for a given platform as a container
func (d *DockerBuild) Image(
	ctx context.Context,
	// the platform of the docker image to return
	// +optional
	// +default="linux/amd64"
	platform dagger.Platform,
) (*dagger.Container, error) {
	// Only exists currently as maps are not supported
	for _, build := range d.Builds {
		pform, err := build.Platform(ctx)
		if err != nil {
			return nil, err
		}

		if pform == platform {
			return build, nil
		}
	}

	return nil, fmt.Errorf("no built image exists for platform '%s'", platform)
}

// Publish the built image to a target registry. Supports publishing of mulit-platform images
func (d *DockerBuild) Publish(
	ctx context.Context,
	// a fully qualified image reference without tags
	// +required
	ref string,
	// a list of tags that should be published with the image
	// +optional
	// +default=["latest"]
	tags []string,
) (string, error) {
	// Sanitise the ref, stripping off any tags or trailing forward slashes that may
	// have accidentally been included due to dynamic CI variables
	imgRef := strings.TrimRight(ref, ":/")

	ctr := dag.Container()
	if d.Auth != nil {
		ctr = ctr.WithRegistryAuth(d.Auth.Registry, d.Auth.Username, d.Auth.Password)
	}

	var imageRefs []string
	for _, tag := range tags {
		if idx := strings.LastIndex(tag, "/"); idx > -1 {
			tag = tag[idx+1:]
		}

		imageRef, err := ctr.Publish(
			ctx,
			fmt.Sprintf("%s:%s", imgRef, tag),
			dagger.ContainerPublishOpts{
				PlatformVariants:  d.Builds,
				ForcedCompression: dagger.Gzip,
			},
		)
		if err != nil {
			return "", err
		}
		imageRefs = append(imageRefs, imageRef)
	}

	return strings.Join(imageRefs, "\n"), nil
}
