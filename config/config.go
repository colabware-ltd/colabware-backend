package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	DBUser        string  `mapstructure:"DB_USER"`
	DBPass        string  `mapstructure:"DB_PASS"`
	DBAddr        string  `mapstructure:"DB_ADDR"`
	StripeKey     string  `mapstructure:"STRIPE_KEY"`
	GitHubCID     string  `mapstructure:"GITHUB_CID"`
	GitHubCSecret string  `mapstructure:"GITHUB_CSECRET"`
	EthNode       string  `mapstructure:"ETH_NODE"`
	EthNodeWSS    string  `mapstructure:"ETH_NODE_WSS"`
	EthKey        string  `mapstructure:"ETH_KEY"`
	EthAddr       string  `mapstructure:"ETH_ADDR"`
	EthChainId    int64   `mapstructure:"ETH_CHAIN_ID"`
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

// ETH_NODE=https://polygon-mumbai.g.alchemy.com/v2/hIpnTe7q_QE-JQGkyuxdkpC9HQABLn9l
// ETH_KEY=5e07055cc82d4df284f65da9296e5ec46010fc2b0061f68d4439a01f09ecbb95
// ETH_CHAIN_ID=80001
