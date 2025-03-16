package conf

import (
	"log"
	"os"

	"github.com/spf13/viper"
)

const namespace_file string = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

// initialize the integration of configs
func InitializeConfig() {

	// setting the default config value, if no other sources are parsed
	log.Println("set default configs")
	for _, item := range configs {
		viper.SetDefault(item.Name, item.Value)
	}

	// set the source of the configs from file and load from it
	log.Println("loading configs from file")
	viper.SetConfigName("echosec")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/opt/echosec")
	if err := viper.ReadInConfig(); err != nil {
		log.Println("unable to read config file", "source", viper.ConfigFileUsed(), "err", err)
	}

	// load configs from env vars
	viper.SetEnvPrefix("ESEC")
	viper.AutomaticEnv()

	// validate if debug is activated
	if viper.GetBool("log.debug") {
		viper.Debug()
		log.Println("currently set configs", "configs", viper.AllSettings())
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
			log.Println("couldn't calculate namespace, are we running in a cluster?", "error", err)
			os.Exit(1)
		} else {
			return []string{string(dat)}
		}
		return []string{}
	}()},
	{Name: "syncperiodminutes", Value: 10},
}
