// Package config provides application configuration and logging setup.
//
// It manages global configuration including logging levels and output
// formatting. Uses viper for configuration management and logrus for
// structured logging.
//
// Example usage:
//
//	// Initialize configuration system
//	config.InitConfig()
//
//	// Set logging level
//	config.SetLogLevel(true, false)  // verbose mode
//
//	// Use global logger
//	config.Logger.Info("Bundle created")
//	config.Logger.Debug("Debug message")
package config

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds the application configuration.
//
// It contains runtime settings for logging and output control.
//
// Fields:
//   - LogLevel: logging level (debug, info, error)
//   - Verbose: enable verbose/debug output
//   - Quiet: suppress non-error output
type Config struct {
	LogLevel string
	Verbose  bool
	Quiet    bool
}

var (
	// Logger is the global logrus instance.
	//
	// Use this for all logging throughout the application:
	//
	//	config.Logger.Info("Operation completed")
	//	config.Logger.WithField("path", bundlePath).Debug("Loading bundle")
	//	config.Logger.Error("Failed to create bundle")
	Logger = logrus.New()
)

// InitConfig initializes the configuration system.
//
// It sets default values and configures the global logger with appropriate
// formatting for CLI output. Also sets up configuration file locations.
//
// Example:
//
//	func main() {
//	    config.InitConfig()
//	    config.Logger.Info("Application started")
//	}
func InitConfig() {
	viper.SetDefault("log_level", "info")
	
	// Setup logrus first so we can log config loading
	Logger.SetOutput(os.Stderr)
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	Logger.SetLevel(logrus.InfoLevel) // Default to info until config is loaded
	
	// Set configuration file name and locations
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.config/bundle")
	viper.AddConfigPath("/etc/bundle")
	viper.AddConfigPath(".")
	
	Logger.Debugf("Configuration search paths:")
	Logger.Debugf("  - $HOME/.config/bundle/config.yaml")
	Logger.Debugf("  - /etc/bundle/config.yaml")
	Logger.Debugf("  - ./config.yaml")
	
	// Read configuration file (ignore if not found)
	err := viper.ReadInConfig()
	if err != nil {
		Logger.Debugf("No configuration file found: %v", err)
		Logger.Debugf("Using default configuration")
	} else {
		Logger.Infof("Configuration loaded from: %s", viper.ConfigFileUsed())
		Logger.Debugf("Configuration content:")
		
		// Log all configuration keys and values
		allSettings := viper.AllSettings()
		for key, value := range allSettings {
			Logger.Debugf("  %s = %v", key, value)
		}
		
		// Set log level from config
		logLevel := viper.GetString("log_level")
		if logLevel == "debug" {
			Logger.SetLevel(logrus.DebugLevel)
			Logger.Debugf("Log level set to debug from configuration")
		}
	}
	
	viper.AutomaticEnv()
}

// SetLogLevel configures the logging level.
//
// It adjusts the global logger based on verbosity flags:
//   - verbose: debug level (shows all messages)
//   - quiet: error level (only errors)
//   - normal: info level (informational messages)
//
// Example:
//
//	// Verbose mode
//	config.SetLogLevel(true, false)
//	config.Logger.Debug("This will be shown")
//
//	// Quiet mode
//	config.SetLogLevel(false, true)
//	config.Logger.Info("This will be hidden")
//	config.Logger.Error("This will be shown")
//
// Parameters:
//   - verbose: enable debug-level logging
//   - quiet: only show errors
func SetLogLevel(verbose, quiet bool) {
	if verbose {
		Logger.SetLevel(logrus.DebugLevel)
	} else if quiet {
		Logger.SetLevel(logrus.ErrorLevel)
	} else {
		Logger.SetLevel(logrus.InfoLevel)
	}
}
