package flags

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Load loads flags from the config file and environment variables
func Load(cmd *cobra.Command, args []string) error {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	// alias k6-cloud-project to k6-cloud-project-id to use K6's default environment variable name
	v.RegisterAlias("k6-cloud-project-id", "k6-cloud-project")
	
	cfgFile, err := cmd.Flags().GetString("config")
	if err != nil {
	    return fmt.Errorf("failed to get config flag: %w", err)
	}

	if cfgFile != "" {
	    v.SetConfigFile(cfgFile)
	}
	
	err = v.ReadInConfig(); 
	if err != nil {
	    if cmd.Flags().Changed("config") && !errors.Is(err, viper.ConfigFileNotFoundError{}) {	
		return fmt.Errorf("failed to read config file: %w", err)
	    }
	}

	// Bind each flag to its respective config/env value if not explicitly set
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Only bind if the flag wasn't explicitly set
		if !f.Changed {
			configKey := strings.ReplaceAll(f.Name, "-", ".")
			if v.IsSet(configKey) {
				val := v.Get(configKey)
				_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			}
		}
	})

	return nil
}