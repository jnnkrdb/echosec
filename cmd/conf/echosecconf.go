package conf

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/spf13/viper"
)

const namespace_file string = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// load the configs from default or configfile
func LoadConfig(setupLog logr.Logger) {

	setupLog.Info("set default configs")
	setDefaults(setupLog)

	setupLog.Info("loading configs")

	// set the source of the config file
	viper.SetConfigName("echosec")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/opt/echosec")
	if err := viper.ReadInConfig(); err != nil {
		setupLog.Error(err, "unable to read config file", "source", viper.ConfigFileUsed())
	}

	if viper.GetBool("debug") {
		viper.Debug()
	}
}

// set the default configs
func setDefaults(setupLog logr.Logger) {

	// set the default config values
	viper.SetDefault("debug", false)
	viper.SetDefault("namespaces",
		func() []string {
			if dat, err := os.ReadFile(namespace_file); err != nil {
				setupLog.Error(err, "couldn't calculate namespace, are we running in a cluster?")
				os.Exit(1)
			} else {
				return []string{string(dat)}
			}
			return []string{}
		}())
	viper.SetDefault("syncperiodminutes", 10)
	var selectorLabels = make(map[string]string)
	selectorLabels["echosec.jnnkrdb.de/mirror-me"] = "true"
	viper.SetDefault("labels.selector", selectorLabels)
}
