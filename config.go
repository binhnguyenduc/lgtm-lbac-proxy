package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type LogConfig struct {
	Level     int  `mapstructure:"level"`
	LogTokens bool `mapstructure:"log_tokens"`
}

type WebConfig struct {
	ProxyPort           int    `mapstructure:"proxy_port"`
	MetricsPort         int    `mapstructure:"metrics_port"`
	Host                string `mapstructure:"host"`
	TLSVerifySkip       bool   `mapstructure:"tls_verify_skip"`
	TrustedRootCaPath   string `mapstructure:"trusted_root_ca_path"`
	JwksCertURL         string `mapstructure:"jwks_cert_url"`
	OAuthGroupName      string `mapstructure:"oauth_group_name"`
	ServiceAccountToken string `mapstructure:"service_account_token"`
	AuthHeader          string `mapstructure:"auth_header"`
}

type AdminConfig struct {
	Bypass bool   `mapstructure:"bypass"`
	Group  string `mapstructure:"group"`
}

type AlertConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	TokenHeader string `mapstructure:"token_header"`
	CertURL     string `mapstructure:"alert_cert_url"`
	Cert        string `mapstructure:"alert_cert"`
}

type DevConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
}

type ThanosConfig struct {
	URL          string            `mapstructure:"url"`
	TenantLabel  string            `mapstructure:"tenant_label"`
	UseMutualTLS bool              `mapstructure:"use_mutual_tls"`
	Cert         string            `mapstructure:"cert"`
	Key          string            `mapstructure:"key"`
	Headers      map[string]string `mapstructure:"headers"`
	ActorHeader  string            `mapstructure:"actor_header"`
}

type LokiConfig struct {
	URL          string            `mapstructure:"url"`
	TenantLabel  string            `mapstructure:"tenant_label"`
	UseMutualTLS bool              `mapstructure:"use_mutual_tls"`
	Cert         string            `mapstructure:"cert"`
	Key          string            `mapstructure:"key"`
	Headers      map[string]string `mapstructure:"headers"`
	ActorHeader  string            `mapstructure:"actor_header"`
}

type TempoConfig struct {
	URL          string            `mapstructure:"url"`
	TenantLabel  string            `mapstructure:"tenant_label"`
	UseMutualTLS bool              `mapstructure:"use_mutual_tls"`
	Cert         string            `mapstructure:"cert"`
	Key          string            `mapstructure:"key"`
	Headers      map[string]string `mapstructure:"headers"`
	ActorHeader  string            `mapstructure:"actor_header"`
}

type Config struct {
	Log    LogConfig    `mapstructure:"log"`
	Web    WebConfig    `mapstructure:"web"`
	Admin  AdminConfig  `mapstructure:"admin"`
	Alert  AlertConfig  `mapstructure:"alert"`
	Dev    DevConfig    `mapstructure:"dev"`
	Thanos ThanosConfig `mapstructure:"thanos"`
	Loki   LokiConfig   `mapstructure:"loki"`
	Tempo  TempoConfig  `mapstructure:"tempo"`
}

func (a *App) WithConfig() *App {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/config/config/")
	v.AddConfigPath("./configs")
	err := v.MergeInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error no config found")
		return nil
	}
	a.Cfg = &Config{}
	err = v.Unmarshal(a.Cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while unmarshalling config file")
	}
	// Set default auth header if not configured
	if a.Cfg.Web.AuthHeader == "" {
		a.Cfg.Web.AuthHeader = "Authorization"
	}
	// Validate Tempo configuration if provided
	a.validateTempoConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().Str("file", e.Name).Msg("Config file changed")
		err := v.Unmarshal(a.Cfg)
		if err != nil {
			log.Error().Err(err).Msg("Error while unmarshalling config file")
			a.healthy = false
		}
		// Set default auth header if not configured
		if a.Cfg.Web.AuthHeader == "" {
			a.Cfg.Web.AuthHeader = "Authorization"
		}
		// Validate Tempo configuration if provided
		a.validateTempoConfig()
		zerolog.SetGlobalLevel(zerolog.Level(a.Cfg.Log.Level))
	})
	v.WatchConfig()
	zerolog.SetGlobalLevel(zerolog.Level(a.Cfg.Log.Level))
	log.Debug().Any("config", a.Cfg).Msg("")
	return a
}

func (a *App) WithSAT() *App {
	if a.Cfg.Dev.Enabled {
		a.ServiceAccountToken = a.Cfg.Web.ServiceAccountToken
		return a
	}
	sa, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Fatal().Err(err).Msg("Error while reading service account token")
	}
	a.ServiceAccountToken = string(sa)
	return a
}

func (a *App) WithTLSConfig() *App {
	caCert, err := os.ReadFile("/etc/ssl/ca/ca-certificates.crt")
	if err != nil {
		log.Fatal().Err(err).Msg("Error while reading CA certificate")
	}
	log.Trace().Bytes("caCert", caCert).Msg("")

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
		log.Fatal().Msg("Failed to append CA certificate")
	}
	log.Debug().Any("rootCAs", rootCAs).Msg("")

	if a.Cfg.Web.TrustedRootCaPath != "" {
		err := filepath.Walk(a.Cfg.Web.TrustedRootCaPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || strings.Contains(info.Name(), "..") {
				return nil
			}

			certs, err := os.ReadFile(path)
			if err != nil {
				log.Error().Err(err).Msg("Error while reading trusted CA")
				return err
			}
			log.Debug().Str("path", path).Msg("Adding trusted CA")
			certs = append(certs, []byte("\n")...)
			rootCAs.AppendCertsFromPEM(certs)

			return nil
		})
		if err != nil {
			log.Error().Err(err).Msg("Error while traversing directory")
		}
	}

	var certificates []tls.Certificate

	lokiCert, err := tls.LoadX509KeyPair(a.Cfg.Loki.Cert, a.Cfg.Loki.Key)
	if err != nil {
		log.Error().Err(err).Msg("Error while loading loki certificate")
	} else {
		log.Debug().Str("path", a.Cfg.Loki.Cert).Msg("Adding Loki certificate")
		certificates = append(certificates, lokiCert)
	}

	thanosCert, err := tls.LoadX509KeyPair(a.Cfg.Thanos.Cert, a.Cfg.Thanos.Key)
	if err != nil {
		log.Error().Err(err).Msg("Error while loading thanos certificate")
	} else {
		log.Debug().Str("path", a.Cfg.Thanos.Cert).Msg("Adding Thanos certificate")
		certificates = append(certificates, thanosCert)
	}

	tempoCert, err := tls.LoadX509KeyPair(a.Cfg.Tempo.Cert, a.Cfg.Tempo.Key)
	if err != nil {
		log.Error().Err(err).Msg("Error while loading tempo certificate")
	} else {
		log.Debug().Str("path", a.Cfg.Tempo.Cert).Msg("Adding Tempo certificate")
		certificates = append(certificates, tempoCert)
	}

	config := &tls.Config{
		InsecureSkipVerify: a.Cfg.Web.TLSVerifySkip,
		RootCAs:            rootCAs,
		Certificates:       certificates,
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = config
	return a
}

func (a *App) WithJWKS() *App {
	log.Info().Msg("Init JWKS config")
	urls := []string{a.Cfg.Web.JwksCertURL}
	if a.Cfg.Alert.Enabled {
		urls = []string{a.Cfg.Web.JwksCertURL, a.Cfg.Alert.CertURL}
	}
	var cert json.RawMessage
	cert = nil
	if a.Cfg.Alert.Cert != "" {
		cert = json.RawMessage(a.Cfg.Alert.Cert)
	}
	jwks, err := NewCombinedJwks(context.Background(), urls, cert)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create a keyfunc from the server's URL")
	}
	log.Info().Str("url", a.Cfg.Web.JwksCertURL).Msg("JWKS URL")
	a.Jwks = jwks
	return a
}

// validateTempoConfig validates Tempo configuration settings
func (a *App) validateTempoConfig() {
	// Skip validation if Tempo URL is not configured
	if a.Cfg.Tempo.URL == "" {
		return
	}

	// Validate URL format
	if !strings.HasPrefix(a.Cfg.Tempo.URL, "http://") && !strings.HasPrefix(a.Cfg.Tempo.URL, "https://") {
		log.Warn().Str("url", a.Cfg.Tempo.URL).Msg("Tempo URL should start with http:// or https://")
	}

	// Validate tenant_label format (TraceQL attributes must start with scope prefix)
	if a.Cfg.Tempo.TenantLabel != "" {
		validPrefixes := []string{"resource.", "span.", "event.", "link.", "."}
		hasValidPrefix := false
		for _, prefix := range validPrefixes {
			if strings.HasPrefix(a.Cfg.Tempo.TenantLabel, prefix) {
				hasValidPrefix = true
				break
			}
		}
		if !hasValidPrefix {
			log.Warn().
				Str("tenant_label", a.Cfg.Tempo.TenantLabel).
				Msg("Tempo tenant_label should start with resource., span., event., link., or . (for intrinsic attributes)")
		}
	}

	// Validate certificate files exist if mTLS is enabled
	if a.Cfg.Tempo.UseMutualTLS {
		if a.Cfg.Tempo.Cert != "" {
			if _, err := os.Stat(a.Cfg.Tempo.Cert); os.IsNotExist(err) {
				log.Error().Str("cert", a.Cfg.Tempo.Cert).Msg("Tempo certificate file not found")
			}
		}
		if a.Cfg.Tempo.Key != "" {
			if _, err := os.Stat(a.Cfg.Tempo.Key); os.IsNotExist(err) {
				log.Error().Str("key", a.Cfg.Tempo.Key).Msg("Tempo key file not found")
			}
		}
	}

	log.Debug().
		Str("url", a.Cfg.Tempo.URL).
		Str("tenant_label", a.Cfg.Tempo.TenantLabel).
		Bool("use_mutual_tls", a.Cfg.Tempo.UseMutualTLS).
		Msg("Tempo configuration validated")
}
