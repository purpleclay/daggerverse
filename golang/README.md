# Golang

A collection of functions for building, testing and scanning your Go project for vulnerabilities.

To select a Go project, a path must be provided for the `--src` flag. The base image used by all functions is automatically resolved from the project version defined within the `go.mod` file. Auto-detection is supported for Go `1.17` and above:

- `>= 1.17 < 1.20`: the Debian `bullseye` image is used.
- `>= 1.20`: the Debian `bookworm` image is used.

Set the `--image` flag to switch to using your custom image.

## Building a Go binary

Builds a static release binary, assuming `main.go` is in the root directory of the target project. The `GOOS` and `GOARCH` environment variables are resolved to the runtime of the base image:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . build
```

A build can be customized:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . build \
  --os darwin \
  --arch amd64 \
  --main cmd/main.go \
  --out my-binary
```

## Running your unit tests

Executes all unit tests within a target project. Installs both the `tparse` and `gotestsum` packages to generate both a JSON (`test-report.json`) and JUnit (`junit-report.xml`) test report:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . test
```

A test can be customized:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . test \
  --run 'TestSingleFeature' \
  --verbose
```

## Benchmarking performance

Executes all benchmarks within a target project. Results are captured within a report called `bench.out`:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . bench
```

Print out the report contents by:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . bench file --path bench.out contents
```

## Scanning for vulnerabilities

Installs and runs the `golvulncheck` binary against the target project. All output is captured within a report called `vulncheck.out`:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . vulncheck
```

Print out the report contents by:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . vulncheck file --path vulncheck.out contents
```

## Printing the Go version

A utility function that prints the version of Go defined within a `go.mod` file:

```sh
dagger call -m github.com/purpleclay/daggerverse/golang --src . mod-version
```

```sh
1.21.7
```
