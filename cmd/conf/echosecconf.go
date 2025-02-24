package conf

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

const namespace_file string = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// initialize the integration of configs
func InitializeConfig() {

	// setting the default config value, if no other sources are parsed
	slog.Info("set default configs")
	for _, item := range configs {
		viper.SetDefault(item.Name, item.Value)
	}

	// set the source of the configs from file and load from it
	slog.Info("loading configs from file")
	viper.SetConfigName("echosec")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/opt/echosec")
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("unable to read config file", "source", viper.ConfigFileUsed(), "err", err)
	}

	// load configs from env vars
	viper.SetEnvPrefix("ESEC")
	viper.AutomaticEnv()

	// validate if debug is activated
	if viper.GetBool("debug") {
		viper.Debug()
		slog.Info("currently set configs", "configs", viper.AllSettings())
	}
}

// ----------------------------------------------------------------------------------------------------------- Setting the defaults

var configs = []struct {
	Name  string
	Value any
}{
	{Name: "log.debug", Value: false},
	{Name: "namespaces", Value: func() []string {
		if dat, err := os.ReadFile(namespace_file); err != nil {
			slog.Error("couldn't calculate namespace, are we running in a cluster?", "error", err)
			os.Exit(1)
		} else {
			return []string{string(dat)}
		}
		return []string{}
	}()},
	{Name: "syncperiodminutes", Value: 10},
	{Name: "labels.selector", Value: func() map[string]string {
		var selectorLabels = make(map[string]string)
		selectorLabels["echosec.jnnkrdb.de/mirror-me"] = "true"
		return selectorLabels
	}()},
}
