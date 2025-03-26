// Unit tests for the oci-login module

package main

import (
	"context"
	"dagger/oci-login/tests/internal/dagger"
	"fmt"

	"github.com/andreyvit/diff"
	"github.com/sourcegraph/conc/pool"
)

const (
	dockerPassword = "c8H96YDRENibMQ=="
	ghcrPassword   = "6VXzOeygB8KrsQ=="
	quayPassword   = "XOs1cDjkZTHCPA=="

	expectedAuth = `{"auths":{"docker.io":{"auth":"YmF0bWFuOmM4SDk2WURSRU5pYk1RPT0="},"ghcr.io":{"auth":"am9rZXI6NlZYek9leWdCOEtyc1E9PQ=="},"quay.io":{"auth":"cGVuZ3VpbjpYT3MxY0Rqa1pUSENQQT09"}}}`
)

func newOciLogin() *dagger.OciLogin {
	return dag.OciLogin().
		WithAuth("docker.io", "batman", dag.SetSecret("docker-password", dockerPassword)).
		WithAuth("ghcr.io", "joker", dag.SetSecret("ghcr-password", ghcrPassword)).
		WithAuth("quay.io", "penguin", dag.SetSecret("quay-password", quayPassword))
}

type Tests struct{}

func (m *Tests) AllTests(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)
	p.Go(m.AsConfig)
	p.Go(m.AsSecret)

	return p.Wait()
}

func (m *Tests) AsConfig(ctx context.Context) error {
	cfg := newOciLogin().AsConfig()

	actual, err := cfg.Contents(ctx)
	if err != nil {
		return err
	}

	if actual != expectedAuth {
		return fmt.Errorf("generated auth config does not match: %s", diff.LineDiff(actual, expectedAuth))
	}

	return nil
}

func (m *Tests) AsSecret(ctx context.Context) error {
	sec := newOciLogin().AsSecret(dagger.OciLoginAsSecretOpts{Name: "oci-login"})

	actual, err := sec.Plaintext(ctx)
	if err != nil {
		return err
	}

	if actual != expectedAuth {
		return fmt.Errorf("generated auth config does not match: %s", diff.LineDiff(actual, expectedAuth))
	}
	return nil
}
