package main

import "github.com/spf13/viper"

type Config struct {
	DBUser        string `mapstructure:"DB_USER"`
	DBPass        string `mapstructure:"DB_PASS"`
	DBAddr        string `mapstructure:"DB_ADDR"`
	StripeKey     string `mapstructure:"STRIPE_KEY"`
	GitHubCID     string `mapstructure:"GITHUB_CID"`
	GitHubCSecret string `mapstructure:"GITHUB_CSECRET"`
	EthNode       string `mapstructure:"ETH_NODE"`
	EthNodeWSS    string `mapstructure:"ETH_NODE_WSS"`
	EthKey        string `mapstructure:"ETH_KEY"`	
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("dev")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
