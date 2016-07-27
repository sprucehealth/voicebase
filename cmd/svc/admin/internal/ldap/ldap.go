package ldap

import (
	"crypto/tls"
	"fmt"

	"context"

	"github.com/samuel/go-ldap/ldap"
	"github.com/sprucehealth/backend/libs/golog"
)

// Config represents the ocnfigurable LDAP aspects
type Config struct {
	Address   string
	BaseDN    string
	TLSConfig *tls.Config
}

// AuthProvider returns an auth provider backed by LDAP
type AuthProvider struct {
	ldapCli *ldap.Client
	baseDN  string
}

// NewAuthenticationProvider returns an LDAP compatible authentication provider
func NewAuthenticationProvider(cfg *Config) (*AuthProvider, error) {
	var err error
	var ldapCli *ldap.Client
	if cfg.TLSConfig == nil {
		ldapCli, err = ldap.Dial("tcp", cfg.Address)
		if err != nil {
			return nil, err
		}
	} else {
		ldapCli, err = ldap.DialTLS("tcp", cfg.Address, cfg.TLSConfig)
		if err != nil {
			return nil, err
		}
	}
	return &AuthProvider{
		ldapCli: ldapCli,
		baseDN:  cfg.BaseDN,
	}, nil
}

// Authenticate binds the provided username and password against the initialized LDAP client
func (ap *AuthProvider) Authenticate(ctx context.Context, username, password string) (string, error) {
	bindDN := fmt.Sprintf("uid=%s,%s", username, ap.baseDN)
	golog.ContextLogger(ctx).Debugf("Binding with %s:%s", bindDN, password)
	if err := ap.ldapCli.Bind(bindDN, []byte(password)); err != nil {
		return "", err
	}
	return username, nil
}
