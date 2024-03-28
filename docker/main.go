// Manage your docker based projects
//
// A collection of functions for building, saving and publishing your Docker based projects
package main

import (
	"context"
	"dagger/docker/internal/dagger"
	"fmt"
	"strings"

	"github.com/containerd/containerd/platforms"
)

// Docker dagger module
type Docker struct {
	// +private
	// +optional
	Auth *DockerAuth
}

// New initializes the docker dagger module
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
	Builds []*Container
	// +private
	// +optional
	Auth *DockerAuth
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
	// a list of target platforms for cross-compilation
	// +optional
	platform []Platform) *DockerBuild {
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

	if len(platform) == 0 {
		platform = append(platform, Platform(platforms.DefaultString()))
	}

	var builds []*Container
	for _, pform := range platform {
		ctr := dag.Container(dagger.ContainerOpts{Platform: dagger.Platform(pform)})
		if d.Auth != nil {
			ctr = ctr.WithRegistryAuth(d.Auth.Registry, d.Auth.Username, d.Auth.Password)
		}

		ctr = ctr.Build(src, dagger.ContainerBuildOpts{
			BuildArgs:  buildArgs,
			Dockerfile: file,
			Target:     target,
		})

		builds = append(builds, ctr)
	}

	return &DockerBuild{Builds: builds, Auth: d.Auth}
}

// Save the built image as a tarball ready for exporting
func (d *DockerBuild) Save(
	ctx context.Context,
	// a name for the exported tarball, will automatically be suffixed by its platform
	// +optional
	// +default="image"
	name string,
) *Directory {
	dir := dag.Directory()

	for _, build := range d.Builds {
		platform, _ := build.Platform(ctx)

		dir = dir.WithFile(fmt.Sprintf("%s_%s.tar", name, strings.Replace(string(platform), "/", "_", 1)),
			build.AsTarball(dagger.ContainerAsTarballOpts{
				ForcedCompression: dagger.Gzip,
			}))
	}

	return dir
}

// Publish the built image to a target registry
func (d *DockerBuild) Publish(
	ctx context.Context,
	// a fully qualified image reference without tags
	// +required
	ref string,
	// a list of tags that should be published with the image
	// +optional
	// default="latest"
	tags []string) (string, error) {
	// Sanitise the ref, stripping off any tags that may have accidentally been included
	if strings.LastIndex(ref, ":") > -1 {
		ref = ref[:strings.LastIndex(ref, ":")]
	}

	ctr := dag.Container()
	if d.Auth != nil {
		ctr = ctr.WithRegistryAuth(d.Auth.Registry, d.Auth.Username, d.Auth.Password)
	}

	var imageRefs []string
	for _, tag := range tags {
		imageRef, err := ctr.Publish(ctx,
			fmt.Sprintf("%s:%s", ref, tag),
			ContainerPublishOpts{PlatformVariants: d.Builds},
		)
		if err != nil {
			return "", err
		}
		imageRefs = append(imageRefs, imageRef)
	}

	return strings.Join(imageRefs, "\n"), nil
}
