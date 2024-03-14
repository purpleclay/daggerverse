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
)

// NSV dagger module
type Nsv struct {
	// Base is the image used by all nsv dagger functions
	// +private
	Base *Container

	// Src is a directory that contains the projects source code
	// +private
	Src *Directory
}

// New initializes the golang dagger module
func New(
	// a path to a directory containing the source code
	// +required
	src *Directory) *Nsv {
	return &Nsv{Base: base(), Src: src}
}

func base() *Container {
	return dag.Container().
		From("ghcr.io/purpleclay/nsv:v0.7.0")
}

// Prints the next semantic version based on the commit history of your repository
func (n *Nsv) Next(
	ctx context.Context,
	// a list of relative paths of projects to analyze
	// +optional
	paths []string) (string, error) {
	cmd := []string{"next"}
	if len(paths) > 0 {
		cmd = append(cmd, paths...)
	}

	return n.Base.
		WithEnvVariable("TINI_SUBREAPER", "1").
		WithDirectory("/src", n.Src).
		WithWorkdir("/src").
		WithExec(cmd).
		Stdout(ctx)
}
