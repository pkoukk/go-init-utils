package giu

func Manually() {
	configParam := ConfigParams{
		ConfigName: "config",
		ConfigType: "yaml",
		ConfigPath: []string{"."},
		AutoEnv:    true,
	}

	v, err := NewLocalConfig(configParam)
	if err != nil {
		panic(err)
	}

	var giuConfig GiuConfig[any]
	if err := v.Unmarshal(&giuConfig); err != nil {
		panic(err)
	}

}
