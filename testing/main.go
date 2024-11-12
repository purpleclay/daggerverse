package main

import (
	"context"
	"dagger/testing/internal/dagger"
)

type Testing struct{}

// Returns a container that echoes whatever string argument is provided
func (m *Testing) Test(
	ctx context.Context,
	// the directory to run NSV against
	// +required
	src *dagger.Directory,
) (string, error) {
	return dag.Nsv(src, dagger.NsvOpts{LogLevel: dagger.NsvLogLevelDebug}).
		Next(ctx, dagger.NsvNextOpts{
			Paths: []string{"apko"},
			Show:  true,
		})
}
