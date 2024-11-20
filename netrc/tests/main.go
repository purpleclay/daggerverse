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

	p.Go(m.WithLogin)
	p.Go(m.WithFile)
	p.Go(m.WithFileInvalid)

	return p.Wait()
}

func (m *Tests) WithLogin(ctx context.Context) error {
	cfg, err := dag.Netrc(dagger.NetrcOpts{Format: dagger.Compact}).
		WithLogin("github.com", dag.SetSecret("username", "batman"), dag.SetSecret("password", "gotham")).
		AsFile().
		Sync(ctx)
	if err != nil {
		return err
	}

	actual, err := cfg.Contents(ctx)
	if err != nil {
		return err
	}

	expected := "machine github.com login batman password gotham"
	if actual != expected {
		return fmt.Errorf("generated auto-login configuration file does not match:\n%v",
			diff.LineDiff(expected, actual))
	}

	return nil
}

func (m *Tests) WithFile(ctx context.Context) error {
	content := `machine github.com login batman password gotham
machine gitlab.com
login joker
password arkam`

	cfg := dag.Directory().
		WithNewFile(".netrc", content, dagger.DirectoryWithNewFileOpts{Permissions: 0o600}).
		File(".netrc")

	_, err := dag.Netrc(dagger.NetrcOpts{Format: dagger.Compact}).
		WithFile(cfg).
		AsFile().
		Sync(ctx)
	return err
}

func (m *Tests) WithFileInvalid(ctx context.Context) error {
	content := "machine github.com password arkam login bane"

	cfg := dag.Directory().
		WithNewFile(".netrc", content, dagger.DirectoryWithNewFileOpts{Permissions: 0o600}).
		File(".netrc")

	_, err := dag.Netrc(dagger.NetrcOpts{Format: dagger.Compact}).
		WithFile(cfg).
		AsFile().
		Sync(ctx)
	if err == nil {
		return fmt.Errorf("expected error while parsing invalid auto-login configuration file")
	}

	return nil
}
