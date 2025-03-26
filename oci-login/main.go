// Create an OCI registry authentication file for secure authentication.
//
// Generate an OCI registry authentication file that provides a secure authentication
// approach for any tool that performs a login operation. Tools such as apko and helm,
// provide dedicated login commands that write credentials to disk, which Dagger
// ultimately caches. With this approach, a registry authentication file can (and should)
// be mounted as a dedicated secret to avoid caching.
package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"dagger/oci-login/internal/dagger"
)

// When mapped to a JSON file provides a way to control authenticate to an Image
// Registry, https://github.com/containers/image/blob/main/docs/containers-auth.json.5.md
type ContainerAuth struct {
	Auths map[string]Auth `json:"auths"`
}

// Contains a base64 encoded credential for authenticating to an Image Registry
type Auth struct {
	// +private
	Auth string `json:"auth"`
}

// OCI Login dagger module
type OciLogin struct {
	// +private
	Config ContainerAuth
}

// Initializes the OCI Login dagger module
func New() *OciLogin {
	return &OciLogin{
		Config: ContainerAuth{
			Auths: map[string]Auth{},
		},
	}
}

// Configure credentials for authenticating to an image registry. Can be chained to
// configure multiple credentials in a single pass
func (m *OciLogin) WithAuth(
	ctx context.Context,
	// the hostname (e.g. docker.io) or namespace (e.g. quay.io/user/image) of the
	// registry to authenticate with
	// +required
	hostname string,
	// the name of the user to authenticate with
	// +required
	username string,
	// the password for the user to authenticate with
	// +required
	password *dagger.Secret,
) (*OciLogin, error) {
	passwd, err := password.Plaintext(ctx)
	if err != nil {
		return nil, err
	}

	str := fmt.Sprintf("%s:%s", username, passwd)

	m.Config.Auths[hostname] = Auth{
		Auth: base64.StdEncoding.EncodeToString([]byte(str)),
	}
	return m, nil
}

// Generates a JSON representation of the current OCI login configuration as a file
func (m *OciLogin) AsConfig() *dagger.File {
	config, _ := json.Marshal(m.Config)

	return dag.Directory().
		WithNewFile("oci-config.json", string(config), dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File("oci-config.json")
}

// Generates a JSON representation of the current OCI login configuration as a secret
func (m *OciLogin) AsSecret(
	// a name for the generated secret, defaults to oci-config-x, where x
	// is the md5 hash of the config
	// +optional
	name string,
) *dagger.Secret {
	config, _ := json.Marshal(m.Config)

	if name == "" {
		hash := md5.Sum(config)
		name = fmt.Sprintf("oci-config-%s", hex.EncodeToString(hash[:]))
	}

	return dag.SetSecret(name, string(config))
}
