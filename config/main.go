package config

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	LogLevel string
	Verbose  bool
	Quiet    bool
}

var (
	// Logger is the global logrus instance
	Logger = logrus.New()
)

// InitConfig initializes the configuration system
func InitConfig() {
	viper.SetDefault("log_level", "info")
	viper.AutomaticEnv()

	// Setup logrus
	Logger.SetOutput(os.Stderr)
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
}

// SetLogLevel configures the logging level
func SetLogLevel(verbose, quiet bool) {
	if verbose {
		Logger.SetLevel(logrus.DebugLevel)
	} else if quiet {
		Logger.SetLevel(logrus.ErrorLevel)
	} else {
		Logger.SetLevel(logrus.InfoLevel)
	}
}
