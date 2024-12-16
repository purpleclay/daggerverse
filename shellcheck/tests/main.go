package main

import (
	"context"
	"dagger/tests/internal/dagger"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andreyvit/diff"
	"github.com/sourcegraph/conc/pool"
)

var (
	//go:embed testdata/valid.sh
	validScript string

	//go:embed testdata/invalid.sh
	invalidScript string
)

type ShellcheckReportItem struct {
	Code    int    `json:"code"`
	Level   string `json:"level"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func (i ShellcheckReportItem) String() string {
	return fmt.Sprintf("%d:%s:%d:%s", i.Line, i.Level, i.Code, i.Message)
}

type Tests struct{}

func (m *Tests) AllTests(ctx context.Context) error {
	p := pool.New().WithErrors().WithContext(ctx)

	p.Go(m.CheckValidFile)
	p.Go(m.CheckInvalidFile)
	p.Go(m.CheckInvalidFileWithInclude)
	p.Go(m.CheckInvalidFileWithExclude)

	return p.Wait()
}

func (m *Tests) CheckValidFile(ctx context.Context) error {
	dir := dag.Directory().
		WithNewFile("valid.sh", validScript, dagger.DirectoryWithNewFileOpts{Permissions: 0o755})

	_, err := dag.Shellcheck().Check(ctx, dir, dagger.ShellcheckCheckOpts{Paths: []string{"valid.sh"}})
	return err
}

func (m *Tests) CheckInvalidFile(ctx context.Context) error {
	dir := dag.Directory().
		WithNewFile("invalid.sh", invalidScript, dagger.DirectoryWithNewFileOpts{Permissions: 0o755})

	opts := dagger.ShellcheckCheckOpts{
		Format: "json",
		Paths:  []string{"invalid.sh"},
	}

	_, err := dag.Shellcheck().Check(ctx, dir, opts)

	actual := err.Error()
	if idx := strings.Index(actual, "[{"); idx != -1 {
		actual = actual[idx:]
	}

	var checks []ShellcheckReportItem
	if err := json.NewDecoder(strings.NewReader(actual)).Decode(&checks); err != nil {
		return err
	}

	if len(checks) != 2 {
		return fmt.Errorf("shellcheck report should have 2 items but has %d", len(checks))
	}

	if checks[0].String() != "4:warning:3030:In POSIX sh, arrays are undefined." {
		return fmt.Errorf("shellcheck report line does not match:\n%s",
			diff.LineDiff(checks[0].String(), "4:warning:3030:In POSIX sh, arrays are undefined."))
	}

	if checks[1].String() != "5:warning:3054:In POSIX sh, array references are undefined." {
		return fmt.Errorf("shellcheck report line does not match:\n%s",
			diff.LineDiff(checks[1].String(), "5:warning:3054:In POSIX sh, array references are undefined."))
	}

	return nil
}

func (m *Tests) CheckInvalidFileWithInclude(ctx context.Context) error {
	dir := dag.Directory().
		WithNewFile("invalid.sh", invalidScript, dagger.DirectoryWithNewFileOpts{Permissions: 0o755})

	opts := dagger.ShellcheckCheckOpts{
		Format:  "json",
		Include: []string{"3030"},
		Paths:   []string{"invalid.sh"},
	}

	_, err := dag.Shellcheck().Check(ctx, dir, opts)

	actual := err.Error()
	if idx := strings.Index(actual, "[{"); idx != -1 {
		actual = actual[idx:]
	}

	var checks []ShellcheckReportItem
	if err := json.NewDecoder(strings.NewReader(actual)).Decode(&checks); err != nil {
		return err
	}

	if len(checks) != 1 {
		return fmt.Errorf("shellcheck report should have 1 item but has %d", len(checks))
	}

	if checks[0].Code != 3030 {
		return fmt.Errorf("shellcheck report line does not match:\n%s",
			diff.LineDiff(checks[0].String(), "4:warning:3030:In POSIX sh, arrays are undefined."))
	}

	return nil
}

func (m *Tests) CheckInvalidFileWithExclude(ctx context.Context) error {
	dir := dag.Directory().
		WithNewFile("invalid.sh", invalidScript, dagger.DirectoryWithNewFileOpts{Permissions: 0o755})

	opts := dagger.ShellcheckCheckOpts{
		Exclude: []string{"3054"},
		Format:  "json",
		Paths:   []string{"invalid.sh"},
	}

	_, err := dag.Shellcheck().Check(ctx, dir, opts)

	actual := err.Error()
	if idx := strings.Index(actual, "[{"); idx != -1 {
		actual = actual[idx:]
	}

	var checks []ShellcheckReportItem
	if err := json.NewDecoder(strings.NewReader(actual)).Decode(&checks); err != nil {
		return err
	}

	if len(checks) != 1 {
		return fmt.Errorf("shellcheck report should have 1 item but has %d", len(checks))
	}

	if checks[0].Code != 3030 {
		return fmt.Errorf("shellcheck report line does not match:\n%s",
			diff.LineDiff(checks[0].String(), "4:warning:3030:In POSIX sh, arrays are undefined."))
	}

	return nil
}
