// Kubeconform is a streamlined tool for validating Kubernetes manifests and custom resource definitions (CRD).
//
// Kubeconform can help avoid mistakes and keep Kubernetes setups smooth and trouble-free. It's designed for high performance,
// and uses a self-updating fork of the schemas registry to ensure up-to-date schemas. It supports both YAML and JSON
// manifest files. It can handle CRDs too.

package main

import (
	"context"
	"dagger/kubeconform/internal/dagger"
	_ "embed"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	KubeconformGithubRepo         = "yannh/kubeconform"
	KubeconformBaseImage          = "ghcr.io/yannh/kubeconform"
	KubeconformWorkDir            = "/work"
	KubeconformCRDFileFormat      = "{fullgroup}/{kind}_{version}"
	KubeconformSchemaDir          = "schemas"
	KubeconformSchemaLocationTmpl = "schemas/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json"
)

//go:embed openapi2jsonschema.py
var openapi2JsonSchema string

// Kubeconform dagger module
type Kubeconform struct {
	// +private
	Base *dagger.Container

	// +private
	// +optional
	Schemas *dagger.Directory
}

// Initializes the Kubeconform dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of kubeconform
	// +optional
	base *dagger.Container,
) (*Kubeconform, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"kubeconform", "-v"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	return &Kubeconform{Base: base}, err
}

func defaultImage(ctx context.Context) (*dagger.Container, error) {
	tag, err := dag.Github().GetLatestRelease(KubeconformGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	// The default base image stores the kubeconform binary under /
	return dag.Container().
		From(fmt.Sprintf("%s:v%s", KubeconformBaseImage, tag[1:])).
		WithEnvVariable("PATH", "${PATH}:/", dagger.ContainerWithEnvVariableOpts{Expand: true}), nil
}

// Generates OpenAPI JSON schemas from the provided local Kubernetes CRDs and adds them as
// a schema location to the kubeconform base image. Schemas are generated using the same
// directory structure as https://github.com/datreeio/CRDs-catalog
func (m *Kubeconform) WithLocalCRDs(
	ctx context.Context,
	// a list of paths to local Kubernetes CRD files to transform
	// +required
	crds []*dagger.File,
) (*Kubeconform, error) {
	schemas, err := generateSchemas(ctx, crds)
	if err != nil {
		return m, err
	}

	if m.Schemas == nil {
		m.Schemas = dag.Directory()
	}

	m.Schemas = m.Schemas.WithDirectory(KubeconformSchemaDir, schemas)
	return m, nil
}

func generateSchemas(ctx context.Context, crds []*dagger.File) (*dagger.Directory, error) {
	generator := dag.Container().
		From("python:3.13.0-alpine3.20").
		WithExec([]string{"pip", "install", "--no-cache-dir", "pyaml==24.9.0"}).
		WithEnvVariable("FILENAME_FORMAT", KubeconformCRDFileFormat).
		WithWorkdir(KubeconformWorkDir).
		WithNewFile(
			"openapi2jsonschema.py",
			openapi2JsonSchema,
			dagger.ContainerWithNewFileOpts{
				Permissions: 0o755,
			})

	excludeNames := []string{}
	for _, crd := range crds {
		name, err := crd.Name(ctx)
		if err != nil {
			return nil, err
		}

		generator = generator.
			WithFile(name, crd, dagger.ContainerWithFileOpts{Permissions: 0o644}).
			WithExec([]string{"python3", "openapi2jsonschema.py", name})

		excludeNames = append(excludeNames, name)
	}

	return generator.
		Directory(".").
		WithoutFiles(append(excludeNames, "openapi2jsonschema.py")), nil
}

// Generates OpenAPI JSON schemas from the provided remote Kubernetes CRDs and adds them as
// a schema location to the kubeconform base image. Schemas are generated using the same
// directory structure as https://github.com/datreeio/CRDs-catalog
func (m *Kubeconform) WithRemoteCRDs(
	ctx context.Context,
	// a list of URLs to remote Kubernetes CRD files to transform
	// +required
	crds []string,
) (*Kubeconform, error) {
	// TODO: both WithRemoteCRDs and WithLocalCRDs can be combined with: https://github.com/dagger/dagger/issues/6957
	fetched := []*dagger.File{}
	for _, crd := range crds {
		fetched = append(fetched, dag.HTTP(crd))
	}

	schemas, err := generateSchemas(ctx, fetched)
	if err != nil {
		return m, err
	}

	if m.Schemas == nil {
		m.Schemas = dag.Directory()
	}

	m.Schemas = m.Schemas.WithDirectory(KubeconformSchemaDir, schemas)
	return m, nil
}

// Check and validate your Kubernertes manifests for conformity against the Kubernetes
// OpenAPI specification. This flexibility, allows your manifests to be easily validated
// against different Kubernetes versions. Includes support for validating against CRDs
func (m *Kubeconform) Validate(
	ctx context.Context,
	// a path to a directory containing Kubernetes manifests (YAML and JSON) for validation
	// +optional
	dirs []*dagger.Directory,
	// skip files with missing schemas instead of failing
	// +optional
	ignoreMissingSchemas bool,
	// disable verification of the server's SSL certificate
	// +optional
	insecureSkipTlsVerify bool,
	// the version of kubernertes to validate against, e.g. 1.31.0
	// +optional
	// +default="master"
	kubernetesVersion string,
	// the number of goroutines to run concurrently during validation
	// +optional
	// +default=4
	goroutines int,
	// a path to a Kubernetes manifest file (YAML or JSON) for validation
	// +optional
	files []*dagger.File,
	// a comma-separated list of kinds or GVKs to reject
	// +optional
	reject []string,
	// override the schema search location path
	// +optional
	schemaLocation []string,
	// print results for all resources (verbose)
	// +optional
	show bool,
	// a comma-separated list of kinds or GVKs to ignore
	// +optional
	skip []string,
	// disallow additional properties not in schema or duplicated keys
	// +optional
	strict bool,
	// print a summary at the end
	// +optional
	summary bool,
) (string, error) {
	cmd := []string{"kubeconform"}
	if ignoreMissingSchemas {
		cmd = append(cmd, "-ignore-missing-schemas")
	}

	if insecureSkipTlsVerify {
		cmd = append(cmd, "-insecure-skip-tls-verify")
	}

	if kubernetesVersion != "master" {
		cmd = append(cmd, "-kubernetes-version", kubernetesVersion)
	}

	if goroutines != 4 && goroutines > 0 {
		cmd = append(cmd, "-n", strconv.Itoa(int(goroutines)))
	}

	if len(reject) > 0 {
		cmd = append(cmd, "-reject", strings.Join(reject, ","))
	}

	if len(schemaLocation) > 0 {
		for _, loc := range schemaLocation {
			cmd = append(cmd, "-schema-location", loc)
		}
	}

	if len(skip) > 0 {
		cmd = append(cmd, "-skip", strings.Join(skip, ","))
	}

	if strict {
		cmd = append(cmd, "-strict")
	}

	if summary {
		cmd = append(cmd, "-summary")
	}

	if show {
		cmd = append(cmd, "-verbose")
	}

	ctr := m.Base.WithWorkdir(KubeconformWorkDir)

	if m.Schemas != nil {
		ctr = ctr.WithDirectory(KubeconformWorkDir, m.Schemas)
		cmd = append(cmd, "-schema-location", KubeconformSchemaLocationTmpl)
	}

	counter := 1
	for _, file := range files {
		fname, err := file.Name(ctx)
		if err != nil {
			return "", err
		}

		copyTo := filepath.Join(fmt.Sprintf("%03d", counter), fname)
		cmd = append(cmd, copyTo)

		ctr = ctr.WithFile(copyTo, file, dagger.ContainerWithFileOpts{Permissions: 0o644})
		counter++
	}

	for _, dir := range dirs {
		copyTo := fmt.Sprintf("%03d", counter)
		cmd = append(cmd, copyTo)

		ctr = ctr.WithDirectory(copyTo, dir)
		counter++
	}

	return ctr.WithExec(cmd).Stdout(ctx)
}
