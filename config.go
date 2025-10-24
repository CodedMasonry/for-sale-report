package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// MLS
const MLS_LOGIN_URL = "https://cr.flexmls.com/"
const MLS_SEARCH_URL_BASE = "https://apps.flexmls.com/quick_launch/herald?callback=lookupCallback&_filter="

// Replace {id} with Id and {mlsid} with MLS Id from MLS_SEARCH_URL result
const MLS_SEARCH_HISTORY_URL_BASE = "https://cr.flexmls.com/cgi-bin/mainmenu.cgi?cmd=srv%20srch_rs/detail/addr_hist.html&list_tech_id=x%27{id}%27&srch=Y&ma_search_list=x%27{mlsid}%27"

// FollowUpBoss
const FUB_SYSTEM_HEADER = "ForSaleReport"                 // X-System
const FUB_SYSTEM_KEY = "e50150b78203e92245f6407fdea50dab" // X-System-Key
const FUB_BUFFFER_AMOUNT = 100                            // How many to get per request

// Config represents the application configuration
type Config struct {
	FUB  FUBConfig  `toml:"fub"`
	MLS  MLSConfig  `toml:"mls"`
	SMTP SMTPConfig `toml:"smtp"`
}

// FUBConfig represents FUB-related configuration
type FUBConfig struct {
	APIKey            string   `toml:"api_key"`
	SellerSmartlistID string   `toml:"seller_smartlist_id"`
	ExcludedStages    []string `toml:"excluded_stages"`
}

// MLSConfig represents MLS-related configuration
type MLSConfig struct {
	User string `toml:"user"`
	Pass string `toml:"pass"`
}

// SMTPConfig represents SMTP-related configuration
type SMTPConfig struct {
	User string   `toml:"user"`
	Pass string   `toml:"pass"`
	From string   `toml:"from"`
	To   []string `toml:"to"`
	Host string   `toml:"host"`
	Port string   `toml:"port"`
}

// Global configuration instance
var AppConfig *Config

// getDefaultConfig returns a Config struct with default values
func getDefaultConfig() Config {
	return Config{
		FUB: FUBConfig{
			APIKey:            "",                           // Required - will be empty in default config
			SellerSmartlistID: "",                           // Required - will be empty in default config
			ExcludedStages:    []string{"stage1", "stage2"}, // Example default stages
		},
		MLS: MLSConfig{
			User: "", // Required - will be empty in default config
			Pass: "", // Required - will be empty in default config
		},
		SMTP: SMTPConfig{
			User: "",                           // Required - will be empty in default config
			Pass: "",                           // Required - will be empty in default config
			From: "",                           // Required - will be empty in default config
			To:   []string{"test@example.com"}, // Required - will be empty in default config
			Host: "127.0.0.1",                  // Default value
			Port: "1025",                       // Default value
		},
	}
}

// generateDefaultConfigFile creates a default TOML config file at the specified path
func generateDefaultConfigFile(configPath string) error {
	defaultConfig := getDefaultConfig()

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Write a comment header
	header := `# Application Configuration File
# Fill in the required fields below and customize as needed

`
	if _, err := file.WriteString(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Encode the default config to TOML
	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(defaultConfig); err != nil {
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}

	return nil
}

// loadConfig loads configuration from a TOML file
func loadConfig(configPath string) (*Config, error) {
	var config Config

	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}

// validateConfig checks that all required fields are present
func validateConfig(config *Config) error {
	var missingFields []string

	// Check FUB required fields
	if config.FUB.APIKey == "" {
		missingFields = append(missingFields, "fub.api_key")
	}
	if config.FUB.SellerSmartlistID == "" {
		missingFields = append(missingFields, "fub.seller_smartlist_id")
	}
	if len(config.FUB.ExcludedStages) == 0 {
		missingFields = append(missingFields, "fub.excluded_stages")
	}

	// Check MLS required fields
	if config.MLS.User == "" {
		missingFields = append(missingFields, "mls.user")
	}
	if config.MLS.Pass == "" {
		missingFields = append(missingFields, "mls.pass")
	}

	// Check SMTP required fields
	if config.SMTP.User == "" {
		missingFields = append(missingFields, "smtp.user")
	}
	if config.SMTP.Pass == "" {
		missingFields = append(missingFields, "smtp.pass")
	}
	if config.SMTP.From == "" {
		missingFields = append(missingFields, "smtp.from")
	}
	if len(config.SMTP.To) == 0 {
		missingFields = append(missingFields, "smtp.to")
	}

	// Note: cert is optional, so we don't validate it as required
	// If you want to make it required, uncomment the following:
	// if config.SMTP.Cert == "" {
	//     missingFields = append(missingFields, "smtp.cert")
	// }

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required configuration fields: %s", strings.Join(missingFields, ", "))
	}

	return nil
}

// populateGlobalConfig sets the global AppConfig from the loaded config
func populateGlobalConfig(config *Config) {
	// Trim whitespace from excluded stages
	for i, stage := range config.FUB.ExcludedStages {
		config.FUB.ExcludedStages[i] = strings.TrimSpace(stage)
	}

	// Set the global config
	AppConfig = config
}

// initConfig initializes the configuration from a TOML file
func initConfig() {
	// Parse command line flags
	configPath := flag.String("config", "config.toml", "path to configuration file")
	flag.Parse()

	// Check if config file exists
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Printf("Config file not found at %s, generating default config...\n", *configPath)

		if err := generateDefaultConfigFile(*configPath); err != nil {
			log.Fatalf("Failed to generate default config file: %v", err)
		}

		fmt.Printf("Default config file created at %s\n", *configPath)
		fmt.Println("Please edit the configuration file with your settings and run the application again.")
		os.Exit(0)
	}

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Populate global config
	populateGlobalConfig(config)

	fmt.Printf("Configuration loaded successfully from %s\n", *configPath)
}
