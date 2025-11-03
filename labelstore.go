package main

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
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

	// GetLabelPolicy retrieves the complete label policy for the given user identity.
	// This method supports multi-label enforcement with flexible operators and logic.
	//
	// Parameters:
	//   - identity: User identity containing username and group memberships
	//   - defaultLabel: Default label name to use for policy rules
	//
	// Returns:
	//   - *LabelPolicy: Complete policy with rules, logic, and override settings
	//   - error: Error if policy cannot be retrieved or is invalid
	//
	// Behavior:
	//   - Returns nil error and policy with cluster-wide access for #cluster-wide users
	//   - Returns policy with merged rules from user and group memberships
	//   - Returns error if user has no labels configured
	GetLabelPolicy(identity UserIdentity, defaultLabel string) (*LabelPolicy, error)
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
	rawData     map[string]RawLabelData // Raw YAML data for extended format
	parser      *PolicyParser           // Parser for converting to policies
	policyCache map[string]*LabelPolicy // Cache of parsed policies
}

func (c *FileLabelStore) Connect(config LabelStoreConfig) error {
	// Initialize parser and cache
	c.parser = NewPolicyParser()
	c.policyCache = make(map[string]*LabelPolicy)
	c.rawData = make(map[string]RawLabelData)

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

	// Load raw data for extended format support
	err = c.loadLabels(v)
	if err != nil {
		log.Fatal().Err(err).Msg("Error while loading label configuration")
		return err
	}

	// Watch for configuration changes
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().Str("file", e.Name).Msg("Config file changed")
		err = v.MergeInConfig()
		if err != nil {
			log.Fatal().Err(err).Msg("Error while reloading config file")
		}
		err = c.loadLabels(v)
		if err != nil {
			log.Fatal().Err(err).Msg("Error while reloading label configuration")
		}
	})
	v.WatchConfig()

	log.Debug().Msg("Label store connected")
	return nil
}

// loadLabels loads label configuration from viper (extended format only)
func (c *FileLabelStore) loadLabels(v *viper.Viper) error {
	// Use AllSettings() to preserve case sensitivity of keys from YAML
	// Viper's Unmarshal() lowercases keys by default, which breaks case-sensitive
	// username/group matching. AllSettings() preserves the original case.
	allSettings := v.AllSettings()
	
	rawData := make(map[string]RawLabelData)
	for key, value := range allSettings {
		if data, ok := value.(map[string]interface{}); ok {
			rawData[key] = data
		}
	}

	// Validate format at startup - detect simple format early
	simpleFormatCount := 0
	for key, data := range rawData {
		if _, hasRules := data["_rules"]; !hasRules {
			// Skip cluster-wide entries
			if _, hasClusterWide := data["#cluster-wide"]; hasClusterWide {
				continue
			}
			// Check if this looks like simple format
			if len(data) > 0 {
				simpleFormatCount++
				log.Error().
					Str("entry", key).
					Msg("DEPRECATED FORMAT: Simple label format detected - migration required before v0.12.0+")
			}
		}
	}

	// Fail fast on startup if simple format is detected
	if simpleFormatCount > 0 {
		return fmt.Errorf("DEPRECATED FORMAT: %d entries in simple format detected\n\n"+
			"The simple label format is no longer supported in v0.12.0+\n"+
			"All entries must use the extended format with _rules key.\n\n"+
			"Migration Required:\n"+
			"  1. Run pre-upgrade check: ./scripts/pre-upgrade-check.sh\n"+
			"  2. Use migration tool: cd cmd/migrate-labels && go build && ./migrate-labels -input ../../configs/labels.yaml\n\n"+
			"See cmd/migrate-labels/README.md for detailed instructions", simpleFormatCount)
	}

	c.rawData = rawData

	// Clear policy cache on reload
	c.policyCache = make(map[string]*LabelPolicy)

	log.Debug().Any("rawData", c.rawData).Msg("Labels loaded")
	return nil
}

// GetLabelPolicy retrieves the label policy for a user/group identity.
// It requires the extended YAML format with _rules key.
func (c *FileLabelStore) GetLabelPolicy(identity UserIdentity, defaultLabel string) (*LabelPolicy, error) {
	username := identity.Username
	groups := identity.Groups

	// Check cache first - cache key includes username and groups to handle different group memberships
	cacheKey := username + ":" + strings.Join(groups, ",")
	if cached, ok := c.policyCache[cacheKey]; ok {
		return cached, nil
	}

	// Collect all policies (user + groups)
	var policies []*LabelPolicy

	// User policy
	if userData, ok := c.rawData[username]; ok {
		policy, err := c.parser.ParseUserPolicy(userData, defaultLabel)
		if err != nil {
			log.Warn().Err(err).Str("user", username).Msg("Failed to parse user policy")
		} else {
			policies = append(policies, policy)
		}
	}

	// Group policies
	for _, group := range groups {
		if groupData, ok := c.rawData[group]; ok {
			policy, err := c.parser.ParseUserPolicy(groupData, defaultLabel)
			if err != nil {
				log.Warn().Err(err).Str("group", group).Msg("Failed to parse group policy")
			} else {
				policies = append(policies, policy)
			}
		}
	}

	if len(policies) == 0 {
		return nil, fmt.Errorf("no policy found for user %s", username)
	}

	// Merge policies
	mergedPolicy := c.mergePolicies(policies)

	// Check for cluster-wide access
	if mergedPolicy.HasClusterWideAccess() {
		mergedPolicy = &LabelPolicy{
			Rules: []LabelRule{{Name: "#cluster-wide", Operator: OperatorEquals, Values: []string{"true"}}},
			Logic: LogicAND,
		}
	}

	// Cache the merged policy
	c.policyCache[cacheKey] = mergedPolicy

	return mergedPolicy, nil
}

// mergePolicies combines multiple policies into a single policy.
// Rules are merged with OR logic by default (user can access if any policy allows).
func (c *FileLabelStore) mergePolicies(policies []*LabelPolicy) *LabelPolicy {
	if len(policies) == 0 {
		return &LabelPolicy{Rules: []LabelRule{}, Logic: LogicAND}
	}

	if len(policies) == 1 {
		return policies[0]
	}

	// Merge all rules from all policies
	merged := &LabelPolicy{
		Rules: []LabelRule{},
		Logic: LogicOR, // Multiple policies are combined with OR (more permissive)
	}

	for _, policy := range policies {
		if policy.Override {
			// If a policy has Override=true, it replaces all previous policies
			merged.Rules = policy.Rules
			merged.Logic = policy.Logic
			continue
		}
		merged.Rules = append(merged.Rules, policy.Rules...)
	}

	// Deduplicate rules
	merged.Rules = c.deduplicateRules(merged.Rules)

	return merged
}

// deduplicateRules removes duplicate rules from a slice
func (c *FileLabelStore) deduplicateRules(rules []LabelRule) []LabelRule {
	seen := make(map[string]bool)
	result := []LabelRule{}

	for _, rule := range rules {
		// Create a unique key for the rule
		key := fmt.Sprintf("%s|%s|%v", rule.Name, rule.Operator, rule.Values)
		if !seen[key] {
			seen[key] = true
			result = append(result, rule)
		}
	}

	return result
}
