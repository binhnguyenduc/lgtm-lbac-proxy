package main

import (
	"github.com/rs/zerolog/log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Labelstore defines the interface for retrieving tenant labels based on user identity.
// Implementations are responsible for connecting to their backend and mapping
// user identities to allowed tenant labels.
//
// This interface is authentication-agnostic and focuses purely on the authorization concern
// of mapping user identities to tenant labels.
type Labelstore interface {
	// Connect establishes a connection with the label store using provided configuration.
	// The configuration contains label store-specific settings like file paths or database URLs.
	//
	// Returns error if connection fails or configuration is invalid.
	Connect(config LabelStoreConfig) error

	// GetLabels retrieves tenant labels for the given user identity.
	//
	// Parameters:
	//   - identity: User identity containing username and group memberships
	//
	// Returns:
	//   - map[string]bool: Allowed tenant label values (keys), values always true
	//   - bool: True if user has cluster-wide access (#cluster-wide label)
	//
	// Behavior:
	//   - Returns (nil, true) for cluster-wide access users
	//   - Returns (map, false) for normal users with specific tenant labels
	//   - Returns (empty map, false) if user has no labels configured
	GetLabels(identity UserIdentity) (map[string]bool, bool)
}

// WithLabelStore initializes and connects to the file-based label store.
// It assigns the connected LabelStore to the App instance and returns it.
// If an error occurs during the connection, it logs a fatal error.
func (a *App) WithLabelStore() *App {
	a.LabelStore = &FileLabelStore{}
	err := a.LabelStore.Connect(a.Cfg.LabelStore)
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to labelstore")
	}
	return a
}

type FileLabelStore struct {
	labels map[string]map[string]bool
}

func (c *FileLabelStore) Connect(config LabelStoreConfig) error {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigName("labels")
	v.SetConfigType("yaml")

	// Use configuration paths instead of hardcoded values
	for _, path := range config.ConfigPaths {
		v.AddConfigPath(path)
	}

	err := v.MergeInConfig()
	if err != nil {
		return err
	}
	err = v.Unmarshal(&c.labels)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while unmarshalling config file")
		return err
	}
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().Str("file", e.Name).Msg("Config file changed")
		err = v.MergeInConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("Error while unmarshalling config file")
		}
		err = v.Unmarshal(&c.labels)
		if err != nil {
			log.Fatal().Err(err).Msg("Error while unmarshalling config file")
		}
	})
	v.WatchConfig()
	log.Debug().Any("labels", c.labels).Msg("")
	return nil
}

func (c *FileLabelStore) GetLabels(identity UserIdentity) (map[string]bool, bool) {
	username := identity.Username
	groups := identity.Groups
	mergedNamespaces := make(map[string]bool, len(c.labels[username])*2)
	for k := range c.labels[username] {
		mergedNamespaces[k] = true
		if k == "#cluster-wide" {
			return nil, true
		}
	}
	for _, group := range groups {
		for k := range c.labels[group] {
			mergedNamespaces[k] = true
			if k == "#cluster-wide" {
				return nil, true
			}
		}
	}
	return mergedNamespaces, false
}
