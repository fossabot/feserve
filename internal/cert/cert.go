package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/challenge/tlsalpn01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

const (
	letsencryptStaging    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	letsencryptProduction = "https://acme-v02.api.letsencrypt.org/directory"
)

type Options struct {
	Email     string
	Domains   []string
	CertsPath string
	Debug     bool
}

type Cert struct {
	options *Options
	cert    *certificate.Resource
}

func NewCert(options *Options) *Cert {
	return &Cert{
		options: options,
	}
}

func (c *Cert) Get() *certificate.Resource {
	return c.cert
}

func (c *Cert) ReadFromFile() error {
	cert, err := os.ReadFile(path.Join(c.options.CertsPath, "ssl.cert"))
	if err != nil {
		return err
	}

	key, err := os.ReadFile(path.Join(c.options.CertsPath, "ssl.key"))
	if err != nil {
		return err
	}

	c.cert = &certificate.Resource{
		Certificate: cert,
		PrivateKey:  key,
	}

	return nil
}

func (c *Cert) Generate() error {
	// Create a user. New accounts need an email and private key to start.
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	user := &User{
		Email: c.options.Email,
		key:   privateKey,
	}

	cfg := lego.NewConfig(user)

	if c.options.Debug {
		cfg.CADirURL = letsencryptStaging
	} else {
		cfg.CADirURL = letsencryptProduction
	}
	cfg.Certificate.KeyType = certcrypto.RSA2048

	// A client facilitates communication with the CA server.
	client, err := lego.NewClient(cfg)
	if err != nil {
		return err
	}

	// HTTP port
	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	if err != nil {
		return err
	}

	// TLS Ports
	err = client.Challenge.SetTLSALPN01Provider(tlsalpn01.NewProviderServer("", "443"))
	if err != nil {
		return err
	}

	// New users will need to register
	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}

	user.Registration = reg

	request := certificate.ObtainRequest{
		Domains: c.options.Domains,
		Bundle:  true,
	}

	certificates, err := client.Certificate.Obtain(request)
	if err != nil {
		return err
	}
	c.cert = certificates

	_ = os.Mkdir(c.options.CertsPath, os.ModePerm)

	if err := os.WriteFile(path.Join(c.options.CertsPath, "ssl.cert"), certificates.Certificate, os.ModePerm); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(c.options.CertsPath, "ssl.key"), certificates.PrivateKey, os.ModePerm); err != nil {
		return err
	}

	return nil
}
