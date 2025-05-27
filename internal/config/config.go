package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration settings loaded from the YAML file.
type Config struct {
	GatewayPort         string   `yaml:"gatewayPort"`
	MetricsPort         string   `yaml:"metricsPort"`
	CheckIntervalStr    string   `yaml:"checkInterval"`
	RequestTimeoutStr   string   `yaml:"requestTimeout"`
	RateLimitBackoffStr string   `yaml:"rateLimitBackoff"`
	BlockTolerance      int64    `yaml:"blockTolerance"`
	RpcEndpoints        []string `yaml:"rpcEndpoints"`
	Verbose             bool     `yaml:"verbose"`

	// Parsed values - marked with `yaml:"-"` to be ignored by the parser.
	CheckInterval    time.Duration `yaml:"-"`
	RequestTimeout   time.Duration `yaml:"-"`
	RateLimitBackoff time.Duration `yaml:"-"`
}

// AppConfig holds the global application configuration.
var AppConfig Config

// LoadConfig reads the configuration from the specified YAML file,
// parses it, and sets default values if necessary.
func LoadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	// Unmarshal the YAML data into the AppConfig struct
	err = yaml.Unmarshal(data, &AppConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	// Set defaults if values are missing
	if AppConfig.GatewayPort == "" {
		AppConfig.GatewayPort = ":8545"
	}
	if AppConfig.MetricsPort == "" { // <-- Add default
		AppConfig.MetricsPort = ":9090"
	}
	if AppConfig.CheckIntervalStr == "" {
		AppConfig.CheckIntervalStr = "30s"
	}
	if AppConfig.RequestTimeoutStr == "" {
		AppConfig.RequestTimeoutStr = "5s"
	}
	if AppConfig.RateLimitBackoffStr == "" {
		AppConfig.RateLimitBackoffStr = "1m"
	}
	if AppConfig.BlockTolerance == 0 {
		AppConfig.BlockTolerance = 5
	}
	if len(AppConfig.RpcEndpoints) == 0 {
		return fmt.Errorf("no rpcEndpoints found in config file")
	}

	// Parse duration strings
	AppConfig.CheckInterval, err = time.ParseDuration(AppConfig.CheckIntervalStr)
	if err != nil {
		return fmt.Errorf("invalid checkInterval duration '%s': %w", AppConfig.CheckIntervalStr, err)
	}

	AppConfig.RequestTimeout, err = time.ParseDuration(AppConfig.RequestTimeoutStr)
	if err != nil {
		return fmt.Errorf("invalid requestTimeout duration '%s': %w", AppConfig.RequestTimeoutStr, err)
	}

	AppConfig.RateLimitBackoff, err = time.ParseDuration(AppConfig.RateLimitBackoffStr)
	if err != nil {
		return fmt.Errorf("invalid rateLimitBackoff duration '%s': %w", AppConfig.RateLimitBackoffStr, err)
	}

	fmt.Printf("Configuration loaded successfully from %s.\n", filename)
	return nil
}
