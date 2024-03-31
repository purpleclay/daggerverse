// Build and Publish OCI Container Images from apk packages
package main

import "strings"

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
