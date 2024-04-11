// Manage your docker based projects
//
// A collection of functions for building, saving and publishing your Docker based projects
package main

import (
	"context"
	"dagger/docker/internal/dagger"
	"fmt"
	"strings"
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
//
// Authenticate with a registry:
// `dagger call --registry ghcr.io --username purpleclay --password env:GITHUB_TOKEN`
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
//
// Build an image using the current directory as the context:
// `dagger call build --dir .`
//
// Build an image using cross-compilation:
// `dagger call build --dir . --platfrom "linux/amd64,linux/arm64"`
//
// Build an image using build-args and a build target:
// `dagger call build --dir . --args "VERSION=0.1.0" --target debug`
func (d *Docker) Build(
	// the path to a directory that will be used as the docker context
	// +required
	dir *Directory,
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

	var builds []*Container
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

// Save the built image as a tarball ready for exporting
//
// `dagger call build --dir . save --name awesome_service`
func (d *DockerBuild) Save(
	ctx context.Context,
	// a name for the exported tarball, will automatically be suffixed by its platform (e.g. image_linux_amd64.)
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
			}),
			dagger.DirectoryWithFileOpts{Permissions: 0o644},
		)
	}

	return dir
}

// Retrieves a built image for a given platform
//
// `dagger call build --dir . image`
//
// Cherry-pick a single image from cross compilation:
// `dagger call build --dir . --platform "linux/amd64,linux/arm64" image --platform linux/arm64`
func (d *DockerBuild) Image(
	ctx context.Context,
	// the platform of the docker image to return
	// +optional
	// +default="linux/amd64"
	platform Platform) (*Container, error) {

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

// Publish the built image to a target registry
//
// Publish a built image to the ttl.sh registry:
// `dagger call build --dir . publish --ref ttl.sh/purpleclay-test`
//
// Publish a cross-compiled image to the ttl.sh registry with multiple tags:
// `dagger call build --dir . --platform "linux/amd64,linux/arm64" publish --ref ttl.sh/purpleclay-test --tags "latest,0.1.0"`
func (d *DockerBuild) Publish(
	ctx context.Context,
	// a fully qualified image reference without tags
	// +required
	ref string,
	// a list of tags that should be published with the image
	// +optional
	// +default=["latest"]
	tags []string) (string, error) {
	// Sanitise the ref, stripping off any tags that may have accidentally been included
	if strings.LastIndex(ref, ":") > -1 {
		ref = ref[:strings.LastIndex(ref, ":")]
	}

	if len(tags) == 0 {
		tags = append(tags, "latest")
	}

	ctr := dag.Container()
	if d.Auth != nil {
		ctr = ctr.WithRegistryAuth(d.Auth.Registry, d.Auth.Username, d.Auth.Password)
	}

	var imageRefs []string
	for _, tag := range tags {
		if idx := strings.LastIndex(tag, "/"); idx > -1 {
			tag = tag[idx+1:]
		}

		imageRef, err := ctr.Publish(ctx,
			fmt.Sprintf("%s:%s", ref, tag),
			ContainerPublishOpts{
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
