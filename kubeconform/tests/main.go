package main

import (
	"context"
	"dagger/tests/internal/dagger"
	_ "embed"
	"fmt"
	"strings"

	"github.com/andreyvit/diff"
	"github.com/sourcegraph/conc/pool"
)

var (
	//go:embed testdata/valid.yaml
	valid string

	//go:embed testdata/crd.yaml
	crd string

	//go:embed testdata/invalid.yaml
	invalid string

	//go:embed testdata/function.yaml
	function string

	//go:embed testdata/eventing-crds.yaml
	eventingCRDs string

	//go:embed testdata/serving-crds.yaml
	servingCRDs string
)

type Tests struct{}

func (m *Tests) AllTests(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(m.Validate)
	p.Go(m.ValidateWithCustomSchema)
	p.Go(m.ValidateWithLocalCRDs)
	p.Go(m.ValidateWithRemoteCRDs)
	p.Go(m.ValidateDirectory)
	p.Go(m.ValidateInvalidFile)

	return p.Wait()
}

func (m *Tests) Validate(ctx context.Context) error {
	manifest := dag.Directory().
		WithNewFile("valid.yaml", valid, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("valid.yaml")

	opts := dagger.KubeconformValidateOpts{
		Files:  []*dagger.File{manifest},
		Show:   true,
		Strict: true,
	}

	_, err := dag.Kubeconform().Validate(ctx, opts)
	return err
}

func (m *Tests) ValidateWithCustomSchema(ctx context.Context) error {
	manifest := dag.Directory().
		WithNewFile("crd.yaml", crd, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("crd.yaml")

	opts := dagger.KubeconformValidateOpts{
		Files: []*dagger.File{manifest},
		SchemaLocation: []string{
			"default",
			"https://raw.githubusercontent.com/purpleclay/daggerverse/refs/heads/main/kubeconform/tests/testdata/trainingjob-sagemaker-v1.json",
		},
		Show: true,
	}

	_, err := dag.Kubeconform().Validate(ctx, opts)
	return err
}

func (m *Tests) ValidateWithLocalCRDs(ctx context.Context) error {
	manifest := dag.Directory().
		WithNewFile("function.yaml", function, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("function.yaml")

	evntCRDs := dag.Directory().
		WithNewFile("eventing-crds.yaml", eventingCRDs, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("eventing-crds.yaml")

	srvCRDs := dag.Directory().
		WithNewFile("serving-crds.yaml", servingCRDs, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("serving-crds.yaml")

	opts := dagger.KubeconformValidateOpts{
		Files:          []*dagger.File{manifest},
		SchemaLocation: []string{"default"},
		Show:           true,
	}

	_, err := dag.Kubeconform().
		WithLocalCrds([]*dagger.File{evntCRDs, srvCRDs}).
		Validate(ctx, opts)
	return err
}

func (m *Tests) ValidateWithRemoteCRDs(ctx context.Context) error {
	manifest := dag.Directory().
		WithNewFile("function.yaml", function, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("function.yaml")

	opts := dagger.KubeconformValidateOpts{
		Files:          []*dagger.File{manifest},
		SchemaLocation: []string{"default"},
		Show:           true,
	}

	_, err := dag.Kubeconform().
		WithRemoteCrds([]string{
			"https://github.com/knative/serving/releases/download/knative-v1.15.2/serving-crds.yaml",
			"https://github.com/knative/eventing/releases/download/knative-v1.15.3/eventing-crds.yaml",
		}).
		Validate(ctx, opts)
	return err
}

func (m *Tests) ValidateDirectory(ctx context.Context) error {
	manifests := dag.Directory().
		WithNewFile("valid.yaml", valid, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		WithNewFile("invalid.yaml", invalid, dagger.DirectoryWithNewFileOpts{Permissions: 0o644})

	opts := dagger.KubeconformValidateOpts{
		Dirs:    []*dagger.Directory{manifests},
		Show:    true,
		Summary: true,
	}

	_, err := dag.Kubeconform().Validate(ctx, opts)
	expected := "Summary: 12 resources found in 2 files - Valid: 11, Invalid: 1, Errors: 0, Skipped: 0"

	actual := err.Error()
	if idx := strings.Index(actual, "Summary:"); idx != -1 {
		actual = actual[idx:]
	}

	if actual != expected {
		return fmt.Errorf("kubeconform summary does not match:\n%v",
			diff.LineDiff(expected, actual))
	}

	actual = err.Error()
	if !strings.Contains(actual, "001/valid.yaml") {
		return fmt.Errorf("kubeconform summary does not contain expected file: %s", "001/valid.yaml")
	}

	if !strings.Contains(actual, "001/invalid.yaml") {
		return fmt.Errorf("kubeconform summary does not contain expected file: %s", "001/invalid.yaml")
	}

	return nil
}

func (m *Tests) ValidateInvalidFile(ctx context.Context) error {
	manifest := dag.Directory().
		WithNewFile("invalid.yaml", invalid, dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("invalid.yaml")

	opts := dagger.KubeconformValidateOpts{
		Files:   []*dagger.File{manifest},
		Show:    true,
		Summary: true,
	}

	_, err := dag.Kubeconform().Validate(ctx, opts)
	expected := "Summary: 6 resources found in 1 file - Valid: 5, Invalid: 1, Errors: 0, Skipped: 0"

	actual := err.Error()
	if idx := strings.Index(actual, "Summary:"); idx != -1 {
		actual = actual[idx:]
	}

	if actual != expected {
		return fmt.Errorf("kubeconform summary does not match:\n%v",
			diff.LineDiff(expected, actual))
	}

	return nil
}
