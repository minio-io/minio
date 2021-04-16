// MinIO Cloud Storage, (C) 2015, 2016, 2017, 2018 MinIO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package crypto

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/minio/minio/pkg/kms"
)

var (
	//ErrKMSAuthLogin is raised when there is a failure authenticating to KMS
	ErrKMSAuthLogin = Errorf("Vault service did not return auth info")
)

// VaultKey represents vault encryption key-ring.
type VaultKey struct {
	Name    string `json:"name"`    // The name of the encryption key-ring
	Version int    `json:"version"` // The key version
}

// VaultAuth represents vault authentication type.
// Currently the only supported authentication type is AppRole.
type VaultAuth struct {
	Type    string       `json:"type"`    // The authentication type
	AppRole VaultAppRole `json:"approle"` // The AppRole authentication credentials
}

// VaultAppRole represents vault AppRole authentication credentials
type VaultAppRole struct {
	ID     string `json:"id"`     // The AppRole access ID
	Secret string `json:"secret"` // The AppRole secret
}

// VaultConfig represents vault configuration.
type VaultConfig struct {
	Enabled   bool      `json:"-"`
	Endpoint  string    `json:"endpoint"` // The vault API endpoint as URL
	CAPath    string    `json:"-"`        // The path to PEM-encoded certificate files used for mTLS. Currently not used in config file.
	Auth      VaultAuth `json:"auth"`     // The vault authentication configuration
	Key       VaultKey  `json:"key-id"`   // The named key used for key-generation / decryption.
	Namespace string    `json:"-"`        // The vault namespace of enterprise vault instances
}

// vaultService represents a connection to a vault KMS.
type vaultService struct {
	config        *VaultConfig
	client        *vault.Client
	secret        *vault.Secret
	leaseDuration time.Duration
}

var _ KMS = (*vaultService)(nil) // compiler check that *vaultService implements KMS

// Verify returns a nil error if the vault configuration
// is valid. A valid configuration is either empty or
// contains valid non-default values.
func (v *VaultConfig) Verify() (err error) {
	switch {
	case v.Endpoint == "":
		err = Errorf("crypto: missing hashicorp vault endpoint")
	case strings.ToLower(v.Auth.Type) != "approle":
		err = Errorf("crypto: invalid hashicorp vault authentication type: %s is not supported", v.Auth.Type)
	case v.Auth.AppRole.ID == "":
		err = Errorf("crypto: missing hashicorp vault AppRole ID")
	case v.Auth.AppRole.Secret == "":
		err = Errorf("crypto: missing hashicorp vault AppSecret ID")
	case v.Key.Name == "":
		err = Errorf("crypto: missing hashicorp vault key name")
	case v.Key.Version < 0:
		err = Errorf("crypto: invalid hashicorp vault key version: The key version must not be negative")
	}
	return
}

// NewVault initializes Hashicorp Vault KMS by authenticating
// to Vault with the credentials in config and gets a client
// token for future api calls.
func NewVault(config VaultConfig) (KMS, error) {
	if !config.Enabled {
		return nil, nil
	}
	if err := config.Verify(); err != nil {
		return nil, err
	}

	vaultCfg := vault.Config{Address: config.Endpoint}
	if err := vaultCfg.ConfigureTLS(&vault.TLSConfig{CAPath: config.CAPath}); err != nil {
		return nil, err
	}
	client, err := vault.NewClient(&vaultCfg)
	if err != nil {
		return nil, Errorf("crypto: client error %w", err)
	}
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}
	v := &vaultService{client: client, config: &config}
	if err := v.authenticate(); err != nil {
		return nil, err
	}
	v.renewToken()
	return v, nil
}

// renewToken starts a new go-routine which renews
// the vault authentication token periodically and re-authenticates
// if the token renewal fails
func (v *vaultService) renewToken() {
	retryDelay := v.leaseDuration / 2
	go func() {
		for {
			if v.secret == nil {
				if err := v.authenticate(); err != nil {
					time.Sleep(retryDelay)
					continue
				}
			}
			s, err := v.client.Auth().Token().RenewSelf(int(v.leaseDuration))
			if err != nil || s == nil {
				v.secret = nil
				time.Sleep(retryDelay)
				continue
			}
			if ok, err := s.TokenIsRenewable(); !ok || err != nil {
				v.secret = nil
				continue
			}
			ttl, err := s.TokenTTL()
			if err != nil {
				v.secret = nil
				continue
			}
			v.secret = s
			retryDelay = ttl / 2
			time.Sleep(retryDelay)
		}
	}()
}

// authenticate logs the app to vault, and starts the auto renewer
// before secret expires
func (v *vaultService) authenticate() (err error) {
	payload := map[string]interface{}{
		"role_id":   v.config.Auth.AppRole.ID,
		"secret_id": v.config.Auth.AppRole.Secret,
	}
	var tokenID string
	var ttl time.Duration
	var secret *vault.Secret
	secret, err = v.client.Logical().Write("auth/approle/login", payload)
	if err != nil {
		err = Errorf("crypto: client error %w", err)
		return
	}
	if secret == nil {
		err = ErrKMSAuthLogin
		return
	}

	tokenID, err = secret.TokenID()
	if err != nil {
		err = ErrKMSAuthLogin
		return
	}
	ttl, err = secret.TokenTTL()
	if err != nil {
		err = ErrKMSAuthLogin
		return
	}
	v.client.SetToken(tokenID)
	v.secret = secret
	v.leaseDuration = ttl
	return
}

// Info returns some information about the Vault,
// configuration - like the endpoints or authentication
// method.
func (v *vaultService) Stat() (kms.Status, error) {
	return kms.Status{
		Endpoints:  []string{v.config.Endpoint},
		Name:       "Hashicorp Vault",
		DefaultKey: v.config.Key.Name,
	}, nil
}

// CreateKey is a stub that exists such that the Vault
// client implements the KMS interface. It always returns
// a not-implemented error.
//
// Creating keys requires a KES instance between MinIO and Vault.
func (v *vaultService) CreateKey(keyID string) error {
	// Creating new keys requires KES.
	return errors.New("crypto: creating keys is not supported by Vault")
}

// GenerateKey returns a new plaintext key, generated by the KMS,
// and a sealed version of this plaintext key encrypted using the
// named key referenced by keyID. It also binds the generated key
// cryptographically to the provided context.
func (v *vaultService) GenerateKey(keyID string, ctx Context) (kms.DEK, error) {
	if keyID == "" {
		keyID = v.config.Key.Name
	}
	context, err := ctx.MarshalText()
	if err != nil {
		return kms.DEK{}, err
	}

	payload := map[string]interface{}{
		"context": base64.StdEncoding.EncodeToString(context),
	}
	s, err := v.client.Logical().Write(fmt.Sprintf("/transit/datakey/plaintext/%s", keyID), payload)
	if err != nil {
		return kms.DEK{}, Errorf("crypto: client error %w", err)
	}
	sealKey, ok := s.Data["ciphertext"].(string)
	if !ok {
		return kms.DEK{}, Errorf("crypto: incorrect 'ciphertext' key type %v", s.Data["ciphertext"])
	}

	plainKeyB64, ok := s.Data["plaintext"].(string)
	if !ok {
		return kms.DEK{}, Errorf("crypto: incorrect 'plaintext' key type %v", s.Data["plaintext"])
	}

	plainKey, err := base64.StdEncoding.DecodeString(plainKeyB64)
	if err != nil {
		return kms.DEK{}, Errorf("crypto: invalid base64 key %w", err)
	}
	return kms.DEK{
		KeyID:      keyID,
		Plaintext:  plainKey,
		Ciphertext: []byte(sealKey),
	}, nil
}

// UnsealKey returns the decrypted sealedKey as plaintext key.
// Therefore it sends the sealedKey to the KMS which decrypts
// it using the named key referenced by keyID and responses with
// the plaintext key.
//
// The context must be same context as the one provided while
// generating the plaintext key / sealedKey.
func (v *vaultService) DecryptKey(keyID string, sealedKey []byte, ctx Context) ([]byte, error) {
	context, err := ctx.MarshalText()
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"ciphertext": string(sealedKey),
		"context":    base64.StdEncoding.EncodeToString(context),
	}

	s, err := v.client.Logical().Write(fmt.Sprintf("/transit/decrypt/%s", keyID), payload)
	if err != nil {
		return nil, Errorf("crypto: client error %w", err)
	}

	base64Key, ok := s.Data["plaintext"].(string)
	if !ok {
		return nil, Errorf("crypto: incorrect 'plaintext' key type %v", s.Data["plaintext"])
	}

	plainKey, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, Errorf("crypto: invalid base64 key %w", err)
	}
	return plainKey, nil
}
