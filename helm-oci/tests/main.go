package main

import (
	"context"
	"dagger/tests/internal/dagger"
	"fmt"

	"github.com/andreyvit/diff"
	"github.com/sourcegraph/conc/pool"
)

type Tests struct{}

func (m *Tests) AllTests(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(m.DotEnv)
	p.Go(m.DotEnvGitLab)

	return p.Wait()
}

func (m *Tests) DotEnv(ctx context.Context) error {
	chart := dag.CurrentModule().Source().Directory("./testdata/chart")

	dotenv, err := dag.HelmOci(dagger.HelmOciOpts{Base: dag.Container().From("alpine/helm:3.16.2")}).
		Dotenv(chart, dagger.HelmOciDotenvOpts{Prefix: "TEST_CHART_"}).
		Sync(ctx)
	if err != nil {
		return err
	}

	actual, err := dotenv.Contents(ctx)
	if err != nil {
		return err
	}

	expected := `TEST_CHART_NAME="example"
TEST_CHART_VERSION="0.2.0"
TEST_CHART_APP_VERSION="v0.3.1"
TEST_CHART_KUBE_VERSION=">=1.23.0"
`
	if actual != expected {
		return fmt.Errorf("generated dotenv file does not match:\n%v",
			diff.LineDiff(expected, actual))
	}

	return nil
}

func (m *Tests) DotEnvGitLab(ctx context.Context) error {
	chart := dag.CurrentModule().Source().Directory("./testdata/chart")

	dotenv, err := dag.HelmOci(dagger.HelmOciOpts{Base: dag.Container().From("alpine/helm:3.16.2")}).
		Dotenv(chart, dagger.HelmOciDotenvOpts{Gitlab: true, Prefix: "TEST_CHART_"}).
		Sync(ctx)
	if err != nil {
		return err
	}

	actual, err := dotenv.Contents(ctx)
	if err != nil {
		return err
	}

	expected := `TEST_CHART_NAME=example
TEST_CHART_VERSION=0.2.0
TEST_CHART_APP_VERSION=v0.3.1
TEST_CHART_KUBE_VERSION=>=1.23.0
`
	if actual != expected {
		return fmt.Errorf("generated dotenv file does not match:\n%v",
			diff.LineDiff(expected, actual))
	}

	return nil
}
