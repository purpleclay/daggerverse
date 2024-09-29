// A lightweight wrapper around Helm OCI.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dagger/helm-oci/internal/dagger"

	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

const (
	HelmGithubRepo       = "helm/helm"
	HelmBaseImage        = "alpine/helm"
	HelmRepositoryConfig = "/root/.config/helm/registry/config.json"
	HelmWorkDir          = "/work"
)

// Helm OCI dagger module
type HelmOci struct {
	// +private
	Base *dagger.Container
}

// Initializes the Helm OCI dagger module
func New(
	ctx context.Context,
	// a custom base image containing an installation of helm
	// +optional
	base *dagger.Container,
) (*HelmOci, error) {
	var err error
	if base == nil {
		base, err = defaultImage(ctx)
	} else {
		if _, err = base.WithExec([]string{"helm", "version"}).Sync(ctx); err != nil {
			return nil, err
		}
	}

	base = base.WithUser("root").
		WithoutEnvVariable("HELM_HOME").
		WithoutEnvVariable("HELM_REGISTRY_CONFIG")

	return &HelmOci{Base: base}, err
}

func defaultImage(ctx context.Context) (*dagger.Container, error) {
	tag, err := dag.Github().GetLatestRelease(HelmGithubRepo).Tag(ctx)
	if err != nil {
		return nil, err
	}

	return dag.Container().
		From(fmt.Sprintf("%s:%s", HelmBaseImage, tag[1:])), nil
}

// Packages a chart into a versioned chart archive file using metadata defined within
// the Chart.yaml file. Metadata can be overridden directly with the required flags.
func (m *HelmOci) Package(
	ctx context.Context,
	// a path to the directory containing the Chart.yaml file
	// +required
	dir *dagger.Directory,
	// override the semantic version of the application this chart deploys
	// +optional
	appVersion string,
	// override the semantic version of the chart
	// +optional
	version string,
) (*dagger.File, error) {
	chart, err := resolveChartMetadata(ctx, dir)
	if err != nil {
		return nil, err
	}

	appVer := chart.AppVersion
	if appVersion != "" {
		appVer = appVersion
	}

	ver := chart.Version
	if version != "" {
		ver = version
	}

	return m.Base.
		WithMountedDirectory(HelmWorkDir, dir).
		WithWorkdir(HelmWorkDir).
		WithExec([]string{
			"helm",
			"package",
			".",
			"--app-version",
			appVer,
			"--version",
			ver,
		}).
		File(fmt.Sprintf("%s-%s.tgz", chart.Name, ver)), nil
}

func resolveChartMetadata(ctx context.Context, dir *dagger.Directory) (*chart.Metadata, error) {
	manifest, err := dir.File("Chart.yaml").Contents(ctx)
	if err != nil {
		return nil, err
	}

	metadata := &chart.Metadata{}
	if err := yaml.Unmarshal([]byte(manifest), metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// Push a packaged chart to a chart registry
func (m *HelmOci) Push(
	ctx context.Context,
	// the packaged helm chart
	// +required
	pkg *dagger.File,
	// the OCI registry to publish the chart to, should include full path without chart name
	// +required
	registry string,
	// the username for authenticating with the registry
	// +optional
	username string,
	// the password for authenticating with the registry
	// +optional
	password *dagger.Secret,
) (string, error) {
	regHost, err := extractRegistryHost(registry)
	if err != nil {
		return "", err
	}
	ctr := m.Base

	if username != "" && password != nil {
		helmAuth := dag.OciLogin().WithAuth(regHost, username, password).AsSecret(dagger.OciLoginAsSecretOpts{})
		ctr = ctr.WithMountedSecret(HelmRepositoryConfig, helmAuth)
	}

	reg := registry
	if !strings.HasPrefix(reg, "oci://") {
		reg = fmt.Sprintf("oci://%s", reg)
	}

	tgzName, err := pkg.Name(ctx)
	if err != nil {
		return "", err
	}

	return ctr.
		WithMountedFile(tgzName, pkg).
		WithExec([]string{"helm", "push", tgzName, reg}).
		Stderr(ctx)
}

func extractRegistryHost(registry string) (string, error) {
	reg := strings.TrimPrefix(registry, "oci://")
	idx := strings.Index(reg, "/")
	if idx == -1 {
		return "", fmt.Errorf("malformed registry, could not extract host")
	}
	return reg[:idx], nil
}

// Packages a Helm chart and publishes it to an OCI registry. Semantic versioning for the chart
// is obtained directly from the Chart.yaml file
func (m *HelmOci) PackagePush(
	ctx context.Context,
	// a path to the directory containing the Chart.yaml file
	// +required
	dir *dagger.Directory,
	// override the semantic version of the application this chart deploys
	// +optional
	appVersion string,
	// override the semantic version of the chart
	// +optional
	version string,
	// the OCI registry to publish the chart to, should include full path without chart name
	// +required
	registry string,
	// the username for authenticating with the registry
	// +optional
	username string,
	// the password for authenticating with the registry
	// +optional
	password *dagger.Secret,
) (string, error) {
	pkg, err := m.Package(ctx, dir, appVersion, version)
	if err != nil {
		return "", err
	}

	return m.Push(ctx, pkg, registry, username, password)
}

// Lints a Helm chart
func (m *HelmOci) Lint(
	ctx context.Context,
	// a path to the directory containing the Chart.yaml file
	// +required
	dir *dagger.Directory,
	// fail on any linting errors by returning a non zero exit code
	// +optional
	strict bool,
	// print only warnings and errors
	// +optional
	quiet bool,
) (string, error) {
	cmd := []string{"helm", "lint", "."}

	if strict {
		cmd = append(cmd, "--strict")
	}

	if quiet {
		cmd = append(cmd, "--quiet")
	}

	return m.Base.
		WithMountedDirectory(HelmWorkDir, dir).
		WithWorkdir(HelmWorkDir).
		WithExec(cmd).
		Stdout(ctx)
}

// Renders a chart and captures output to a YAML file. Any values that would
// be looked up within a Kubernetes cluster are faked. When overriding values,
// the priority will always be given to the last (right-most) provided value
func (m *HelmOci) Template(
	ctx context.Context,
	// a path to the directory containing the Chart.yaml file and all templates
	// +required
	dir *dagger.Directory,
	// set values on the command line (can specify multiple or separate values
	// with commas: key1=val1,key2=val2)
	// +optional
	set []string,
	// set values from respective files specified via the command line
	// (can specify multiple or separate values with commas: key1=path1,key2=path2)
	// +optional
	setFile []string,
	// set JSON values on the command line (can specify multiple or separate values
	// with commas: key1=jsonval1,key2=jsonval2)
	// +optional
	setJson []string,
	// set a literal STRING value on the command line (can specify multiple or separate
	// values with commas: key1=val1,key2=val2)
	// +optional
	setLiteral []string,
	// set STRING values on the command line (can specify multiple or separate values
	// with commas: key1=val1,key2=val2)
	// +optional
	setString []string,
	// specify values in a YAML file bundled within the chart directory (can specify multiple)
	// +optional
	values []string,
	// specify values in external YAML files loaded from the file system (can specify multiple).
	// These have a higher precedence over other values files
	// +optional
	valuesExt []*dagger.File) (*dagger.File, error) {
	chart, err := resolveChartMetadata(ctx, dir)
	if err != nil {
		return nil, err
	}

	cmd := []string{"helm", "template", "."}

	cmd = append(cmd, toFlags("--values", values)...)
	// Hand over precedence to the helm CLI directly
	cmd = append(cmd, toFlags("--set", set)...)
	cmd = append(cmd, toFlags("--set-file", setFile)...)
	cmd = append(cmd, toFlags("--set-json", setJson)...)
	cmd = append(cmd, toFlags("--set-literal", setLiteral)...)
	cmd = append(cmd, toFlags("--set-string", setString)...)

	ctr := m.Base.
		WithMountedDirectory(HelmWorkDir, dir).
		WithWorkdir(HelmWorkDir)

	// Ensure values files loaded externally from the chart have higher precedence
	for i, ext := range valuesExt {
		tmpValues := filepath.Join(os.TempDir(), fmt.Sprintf("values-%d.yaml", i+1))
		ctr = ctr.WithFile(tmpValues, ext)
		cmd = append(cmd, "--values", tmpValues)
	}

	template := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s.yaml", strings.ToLower(chart.Name), chart.Version))

	return ctr.
		WithExec(cmd, dagger.ContainerWithExecOpts{RedirectStdout: template}).
		File(template), nil
}

func toFlags(flag string, values []string) []string {
	var flags []string
	for _, v := range values {
		flags = append(flags, flag, v)
	}
	return flags
}
