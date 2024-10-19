// Create a .netrc auto-login configuration file to access remote machines.
//
// Generate a .netrc auto-login configuration file that provides a secure authentication
// approach for transparently authenticating against a remote machine. Tools like Git provide
// out-of-the-box support for a .netrc auto-login configuration file, making accessing
// private repositories painless, especially for languages such as Go. The generated .netrc
// file can (and should) be mounted as a dedicated secret to avoid caching.

package main

import (
	"context"
	"crypto/md5"
	"dagger/netrc/internal/dagger"
	"encoding/hex"
	"fmt"
	"strings"
)

// Holds configuration details for logging into remote sites from a machine
type AutoLogin struct {
	Logins []Login
}

func (a AutoLogin) String() string {
	var buf strings.Builder
	for _, login := range a.Logins {
		buf.WriteString(login.String())
	}
	return strings.TrimSpace(buf.String())
}

// Defines login and initialization information used by the auto-login
// process when connecting to a remote machine
type Login struct {
	// The remote machine name
	Machine string
	// Identifies a user on the remote machine
	Username string
	// Defines a token (or password) used to login into a remote machine
	// as the identified user
	Password string
}

func (l Login) String() string {
	return fmt.Sprintf("machine %s login %s password %s\n", l.Machine, l.Username, l.Password)
}

// Netrc dagger module
type Netrc struct {
	// +private
	Config AutoLogin
}

// Initializes the Netrc dagger module
func New() *Netrc {
	return &Netrc{
		Config: AutoLogin{},
	}
}

// Configures an auto-login configuration for a remote machine with the given credentials.
// Can be chained to configure multiple auto-logins in a single pass
func (m *Netrc) WithLogin(
	ctx context.Context,
	// the remote machine name
	// +required
	machine string,
	// a user on the remote machine that can login
	// +required
	username string,
	// a token (or password) used to login into a remote machine by
	// the identified user
	// +required
	password *dagger.Secret,
) (*Netrc, error) {
	passwd, err := password.Plaintext(ctx)
	if err != nil {
		return nil, err
	}

	login := Login{
		Machine:  machine,
		Username: username,
		Password: passwd,
	}

	m.Config.Logins = append(m.Config.Logins, login)
	return m, nil
}

// Generates and returns a .netrc file based on the current configuration
func (m *Netrc) AsFile() *dagger.File {
	return dag.Directory().
		WithNewFile(".netrc", m.Config.String(), dagger.DirectoryWithNewFileOpts{Permissions: 0o644}).
		File(".netrc")
}

// Generates and returns a .netrc file based on the current configuration that
// can be mounted as a secret to a container
func (m *Netrc) AsSecret(
	// a name for the generated secret, defaults to netrc-x, where x
	// is the md5 hash of the auto-login configuration
	// +optional
	name string,
) *dagger.Secret {
	if name == "" {
		hash := md5.Sum([]byte(m.Config.String()))
		name = fmt.Sprintf("netrc-%s", hex.EncodeToString(hash[:]))
	}

	return dag.SetSecret(name, m.Config.String())
}
