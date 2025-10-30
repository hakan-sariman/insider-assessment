package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

type (
	AppCfg struct {
		Name string `mapstructure:"name"`
		Env  string `mapstructure:"env"`
	}
	ServerCfg struct {
		Port         int           `mapstructure:"port"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
		IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	}
	PostgresCfg struct {
		URL          string `mapstructure:"url"`
		MaxOpenConns int    `mapstructure:"max_open_conns"`
	}
	RedisCfg struct {
		Addr string        `mapstructure:"addr"`
		DB   int           `mapstructure:"db"`
		TTL  time.Duration `mapstructure:"ttl"`
	}
	SchedulerCfg struct {
		Enabled   bool          `mapstructure:"enabled"`
		Interval  time.Duration `mapstructure:"interval"`
		BatchSize int           `mapstructure:"batch_size"`
	}
	OutboundCfg struct {
		URL          string        `mapstructure:"url"`
		Timeout      time.Duration `mapstructure:"timeout"`
		MaxRetries   int           `mapstructure:"max_retries"`
		ExpectStatus int           `mapstructure:"expect_status"`
		AuthHeader   string        `mapstructure:"auth_header"`
		AuthValue    string        `mapstructure:"auth_value"`
	}
	Config struct {
		App       AppCfg       `mapstructure:"app"`
		Server    ServerCfg    `mapstructure:"server"`
		Postgres  PostgresCfg  `mapstructure:"postgres"`
		Redis     RedisCfg     `mapstructure:"redis"`
		Scheduler SchedulerCfg `mapstructure:"scheduler"`
		Outbound  OutboundCfg  `mapstructure:"outbound"`
	}
)

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	if p := os.Getenv("APP_CONFIG_PATH"); p != "" {
		v.SetConfigFile(p)
	}

	v.SetEnvPrefix("APP")
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("app.name", "hakansariman-insider-assessment")
	v.SetDefault("app.env", "dev")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "5s")
	v.SetDefault("server.write_timeout", "5s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("postgres.max_open_conns", 10)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.ttl", "24h")
	v.SetDefault("scheduler.enabled", true)
	v.SetDefault("scheduler.interval", "2m")
	v.SetDefault("scheduler.batch_size", 2)
	v.SetDefault("outbound.timeout", "5s")
	v.SetDefault("outbound.max_retries", 3)
	v.SetDefault("outbound.expect_status", 202)

	if err := v.ReadInConfig(); err != nil {
		// continue with env/defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
