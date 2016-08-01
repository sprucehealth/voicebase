package ldap

import (
	"crypto/tls"
	"fmt"

	"context"

	"github.com/samuel/go-ldap/ldap"
	"github.com/sprucehealth/backend/libs/errors"
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
		golog.Debugf("Initiating connection to LDAP Server at %s", cfg.Address)
		ldapCli, err = ldap.Dial("tcp", cfg.Address)
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else {
		golog.Debugf("Initiating SSL connection to LDAP Server at %s", cfg.Address)
		ldapCli, err = ldap.DialTLS("tcp", cfg.Address, cfg.TLSConfig)
		if err != nil {
			return nil, errors.Trace(err)
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
	if err := ap.ldapCli.Bind(bindDN, []byte(password)); err != nil {
		return "", errors.Trace(err)
	}
	return username, nil
}
