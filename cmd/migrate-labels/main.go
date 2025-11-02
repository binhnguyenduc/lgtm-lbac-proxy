package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// LabelRule represents a single label matching rule
type LabelRule struct {
	Name     string   `yaml:"name"`
	Operator string   `yaml:"operator"`
	Values   []string `yaml:"values"`
}

// ExtendedEntry represents the extended format for a user/group
type ExtendedEntry struct {
	Rules map[string]interface{} `yaml:",inline"`
}

func main() {
	// Setup logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Command line flags
	input := flag.String("input", "", "Path to input labels.yaml file (required)")
	output := flag.String("output", "", "Path to output labels.yaml file (default: input-extended.yaml)")
	defaultLabel := flag.String("default-label", "namespace", "Default label name for simple format conversion")
	dryRun := flag.Bool("dry-run", false, "Print conversion without writing to file")
	validate := flag.Bool("validate", false, "Validate input file without conversion")
	flag.Parse()

	if *input == "" {
		fmt.Println("Usage: migrate-labels -input <file> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Set default output path
	if *output == "" {
		dir := filepath.Dir(*input)
		base := filepath.Base(*input)
		ext := filepath.Ext(base)
		name := base[:len(base)-len(ext)]
		*output = filepath.Join(dir, name+"-extended"+ext)
	}

	// Read input file
	inputData, err := os.ReadFile(*input)
	if err != nil {
		log.Fatal().Err(err).Str("file", *input).Msg("Failed to read input file")
	}

	// Parse YAML
	var data map[string]interface{}
	if err := yaml.Unmarshal(inputData, &data); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse YAML")
	}

	log.Info().Str("file", *input).Int("entries", len(data)).Msg("Loaded label file")

	// Validate mode
	if *validate {
		stats := analyzeFile(data)
		printStats(stats)
		return
	}

	// Convert to extended format
	converted, stats := convertToExtended(data, *defaultLabel)

	// Print statistics
	printStats(stats)

	// Generate YAML output
	outputData, err := yaml.Marshal(converted)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to generate YAML")
	}

	// Dry run mode
	if *dryRun {
		fmt.Println("\n--- Converted YAML (dry run) ---")
		fmt.Println(string(outputData))
		return
	}

	// Write output file
	if err := os.WriteFile(*output, outputData, 0644); err != nil {
		log.Fatal().Err(err).Str("file", *output).Msg("Failed to write output file")
	}

	log.Info().Str("file", *output).Msg("Migration complete")
	fmt.Printf("\n✅ Successfully migrated %d entries to extended format\n", len(data))
	fmt.Printf("📄 Output file: %s\n", *output)
}

type Stats struct {
	TotalEntries    int
	SimpleEntries   int
	ExtendedEntries int
	ClusterWide     int
	Converted       int
	Skipped         int
}

func analyzeFile(data map[string]interface{}) Stats {
	stats := Stats{TotalEntries: len(data)}

	for _, value := range data {
		if m, ok := value.(map[string]interface{}); ok {
			if _, hasRules := m["_rules"]; hasRules {
				stats.ExtendedEntries++
			} else {
				// Check for cluster-wide
				hasClusterWide := false
				for k := range m {
					if k == "#cluster-wide" {
						hasClusterWide = true
						stats.ClusterWide++
						break
					}
				}
				if !hasClusterWide {
					stats.SimpleEntries++
				}
			}
		}
	}

	return stats
}

func convertToExtended(data map[string]interface{}, defaultLabel string) (map[string]interface{}, Stats) {
	result := make(map[string]interface{})
	stats := analyzeFile(data)

	for key, value := range data {
		if m, ok := value.(map[string]interface{}); ok {
			// Already extended format
			if _, hasRules := m["_rules"]; hasRules {
				result[key] = value
				stats.Skipped++
				continue
			}

			// Check for cluster-wide access
			if _, hasClusterWide := m["#cluster-wide"]; hasClusterWide {
				result[key] = value
				stats.Skipped++
				continue
			}

			// Convert simple format to extended
			rules := []LabelRule{}
			var values []string

			for labelKey := range m {
				values = append(values, labelKey)
			}

			if len(values) > 0 {
				rules = append(rules, LabelRule{
					Name:     defaultLabel,
					Operator: "=",
					Values:   values,
				})
			}

			result[key] = map[string]interface{}{
				"_rules": rules,
				"_logic": "AND",
			}
			stats.Converted++
		}
	}

	return result, stats
}

func printStats(stats Stats) {
	fmt.Println("\n--- Migration Statistics ---")
	fmt.Printf("Total entries:     %d\n", stats.TotalEntries)
	fmt.Printf("Simple format:     %d\n", stats.SimpleEntries)
	fmt.Printf("Extended format:   %d\n", stats.ExtendedEntries)
	fmt.Printf("Cluster-wide:      %d\n", stats.ClusterWide)
	if stats.Converted > 0 {
		fmt.Printf("Converted:         %d\n", stats.Converted)
		fmt.Printf("Skipped:           %d\n", stats.Skipped)
	}
}
