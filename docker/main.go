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
	"dagger/docker/internal/dagger"
	"strings"
)

// Docker dagger module
type Docker struct{}

// DockerBuild contains an image built from the provided Dockerfile,
// it serves as an intermediate type for chaining other functions
type DockerBuild struct {
	// +private
	Base *Container
}

// Build an image using a Dockerfile
func (d *Docker) Build(
	// the path to a directory that will be used as the docker context
	// +required
	src *Directory,
	// the path to the Dockfile within the docker context
	// +default="Dockerfile"
	// +required
	file string,
	// a list of build arguments (e.g. arg=value)
	// +optional
	args []string,
	// the name of a target build stage
	// +optional
	target string) *DockerBuild {
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

	con := dag.Container().
		Build(src, dagger.ContainerBuildOpts{
			BuildArgs:  buildArgs,
			Dockerfile: file,
			Target:     target,
		})

	return &DockerBuild{Base: con}
}

// Save the built image as a tarball ready for exporting
func (m *DockerBuild) Save() *File {
	return m.Base.AsTarball()
}

// Publish the built image to a target registry
func (m *DockerBuild) Publish(
	ctx context.Context,
	// the image reference to publish
	// +required
	ref string) (string, error) {
	return m.Base.Publish(ctx, ref)
}
