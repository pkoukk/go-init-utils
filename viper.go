package giu

import "github.com/spf13/viper"

type ConfigParams struct {
	ConfigName string
	ConfigType string
	ConfigPath []string
	AutoEnv    bool
}

var _defaultConfigParams = ConfigParams{
	ConfigName: "config",
	ConfigType: "yaml",
	ConfigPath: []string{"."},
	AutoEnv:    true,
}

func NewLocalConfig(params ConfigParams) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(params.ConfigName)
	if params.ConfigType != "" {
		v.SetConfigType(params.ConfigType)
	}
	if len(params.ConfigPath) > 0 {
		for _, path := range params.ConfigPath {
			v.AddConfigPath(path)
		}
	}
	if params.AutoEnv {
		v.AutomaticEnv()
	}
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}
	return v, nil
}

func DefaultConfig() (*viper.Viper, error) {
	return NewLocalConfig(_defaultConfigParams)
}

type RemoteConfigParams struct {
	Provider   string
	Endpoint   string
	Path       string
	ConfigType string
	AutoEnv    bool
}

func NewConfigFromRemote(params RemoteConfigParams) (*viper.Viper, error) {
	v := viper.New()
	if err := v.AddRemoteProvider(params.Provider, params.Endpoint, params.Path); err != nil {
		return nil, err
	}
	if params.ConfigType != "" {
		v.SetConfigType(params.ConfigType)
	}
	if params.AutoEnv {
		v.AutomaticEnv()
	}
	return v, nil
}
