package main

import (
	"github.com/rs/zerolog/log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Labelstore represents an interface defining methods for connecting to a
// label store and retrieving labels associated with a given OAuth token.
type Labelstore interface {
	// Connect establishes a connection with the label store using App configuration.
	Connect(App) error
	// GetLabels retrieves labels associated with the provided OAuth token.
	// Returns a map containing the labels and a boolean indicating whether
	// the label is cluster-wide or not.
	GetLabels(token OAuthToken) (map[string]bool, bool)
}

// WithLabelStore initializes and connects to the ConfigMap-based label store.
// It assigns the connected LabelStore to the App instance and returns it.
// If an error occurs during the connection, it logs a fatal error.
func (a *App) WithLabelStore() *App {
	a.LabelStore = &ConfigMapHandler{}
	err := a.LabelStore.Connect(*a)
	if err != nil {
		log.Fatal().Err(err).Msg("Error connecting to labelstore")
	}
	return a
}

type ConfigMapHandler struct {
	labels map[string]map[string]bool
}

func (c *ConfigMapHandler) Connect(_ App) error {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.SetConfigName("labels")
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/config/labels/")
	v.AddConfigPath("./configs")
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

func (c *ConfigMapHandler) GetLabels(token OAuthToken) (map[string]bool, bool) {
	username := token.PreferredUsername
	groups := token.Groups
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
