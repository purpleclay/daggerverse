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
	"unicode"

	"github.com/purpleclay/chomp"
)

const (
	machineIdent  = "machine"
	loginIdent    = "login"
	passwordIdent = "password"
)

// Supported formats for generating the auto-login configuration file
type Format string

const (
	// A compact single line format
	Compact Format = "compact"

	// A multiline format
	Full Format = "full"
)

// Holds configuration details for logging into remote sites from a machine
type AutoLogin struct {
	Logins []Login
	Format Format
}

func (a AutoLogin) String() string {
	var buf strings.Builder

	var fmt func(Login) string
	switch a.Format {
	case Compact:
		fmt = compact
	case Full:
		fmt = full
	}

	for _, login := range a.Logins {
		buf.WriteString(fmt(login))
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

func compact(l Login) string {
	return fmt.Sprintf("machine %s login %s password %s\n", l.Machine, l.Username, l.Password)
}

func full(l Login) string {
	return fmt.Sprintf("machine %s\nlogin %s\npassword %s\n", l.Machine, l.Username, l.Password)
}

// Netrc dagger module
type Netrc struct {
	// +private
	Config AutoLogin
}

// Initializes the Netrc dagger module
func New(
	// the format when generating the auto-login configuration file (compact,full)
	// +default="compact"
	format Format,
) *Netrc {
	return &Netrc{
		Config: AutoLogin{Format: format},
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
	username *dagger.Secret,
	// a token (or password) used to login into a remote machine by
	// the identified user
	// +required
	password *dagger.Secret,
) (*Netrc, error) {
	passwd, err := password.Plaintext(ctx)
	if err != nil {
		return nil, err
	}

	uname, err := username.Plaintext(ctx)
	if err != nil {
		return nil, err
	}

	login := Login{
		Machine:  machine,
		Username: uname,
		Password: passwd,
	}

	m.Config.Logins = append(m.Config.Logins, login)
	return m, nil
}

// Loads an existing auto-login configuration from a file. Can be chained to load multiple
// configuration files in a single pass
func (m *Netrc) WithFile(
	ctx context.Context,
	// an existing auto-login configuration file
	// +required
	cfg *dagger.File,
) (*Netrc, error) {
	config, err := cfg.Contents(ctx)
	if err != nil {
		return nil, err
	}

	logins, err := fromConfiguration(config)
	if err != nil {
		return nil, err
	}

	m.Config.Logins = append(m.Config.Logins, logins...)
	return m, nil
}

func fromConfiguration(cfg string) ([]Login, error) {
	_, ext, err := chomp.Map(
		chomp.ManyN(
			chomp.All(
				eatIdent(machineIdent),
				eatIdent(loginIdent),
				eatIdent(passwordIdent),
			), 1),
		func(in []string) []Login {
			// comes in a series of three: (machine, login, password)
			var logins []Login
			for i := 0; i < len(in); i += 3 {
				logins = append(logins, Login{
					Machine:  in[i],
					Username: in[i+1],
					Password: in[i+2],
				})
			}
			return logins
		})(cfg)

	return ext, err
}

type isWhitespace struct{}

func (isWhitespace) Match(r rune) bool {
	return unicode.IsSpace(r)
}

func (isWhitespace) String() string {
	return "is_whitespace"
}

var IsWhitespace = isWhitespace{}

func eatIdent(ident string) chomp.Combinator[string] {
	return func(s string) (string, string, error) {
		rem, ext, err := chomp.All(
			chomp.Tag(ident),
			chomp.While(IsWhitespace),
			chomp.Not("\r\n "),
			chomp.Opt(chomp.While(IsWhitespace)),
			chomp.Opt(chomp.Crlf()),
		)(s)
		if err != nil {
			return rem, "", err
		}

		return rem, ext[2], nil
	}
}

// Generates and returns a .netrc file based on the current configuration
func (m *Netrc) AsFile() *dagger.File {
	return dag.Directory().
		WithNewFile(".netrc", m.Config.String(), dagger.DirectoryWithNewFileOpts{Permissions: 0o600}).
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
