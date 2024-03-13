# Trivy

A collection of functions for scanning container images and source code for vulnerabilities. Set the `--image` flag to switch to using your custom image.

## Scanning an Image

Scans an image from a repository:

```sh
dagger call -m github.com/purpleclay/daggerverse/trivy image --ref golang:1.21.7-bookworm
```

## Scanning an Exported Image

Scan an exported image (.tar) for any vulnerabilities:

```sh
dagger call -m github.com/purpleclay/daggerverse/trivy image-local --ref image.tar
```

## Scanning the Filesystem

Scans the given path for any vulnerabilities:

```sh
dagger call -m github.com/purpleclay/daggerverse/trivy filesystem --ref .
```
