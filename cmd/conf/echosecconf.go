package conf

import (
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
)

func LoadConfig(setupLog logr.Logger) {

	setupLog.Info("loading configs")

	// set the default config values
	viper.SetDefault("debug", false)
	viper.SetDefault("namespaces", []string{""})
	viper.SetDefault("syncperiodminutes", 10)
	var selectorLabels = make(map[string]string)
	selectorLabels["echosec.jnnkrdb.de/mirror-me"] = "true"
	viper.SetDefault("labels.selector", selectorLabels)

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
