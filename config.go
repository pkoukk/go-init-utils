package giu

type GiuConfig[ExtendParams any] struct {
	Logger         map[string]*LoggerParams         `mapstructure:"logger"`
	GormConfig     *GormConfigParams                `mapstructure:"gorm_config"`
	GormConnection map[string]*GormConnectionParams `mapstructure:"gorm_connection"`
	Redis          map[string]*RedisParams          `mapstructure:"redis"`
	Extend         ExtendParams                     `mapstructure:"extend"`
}
