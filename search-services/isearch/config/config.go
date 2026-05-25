package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	LogLevel     string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"DEBUG"`
	IndexTTL     time.Duration `yaml:"index_ttl" env:"INDEX_TTL" env-default:"20s"`
	Address      string        `yaml:"isearch_address" env:"ISEARCH_ADDRESS" env-default:"localhost:80"`
	WordsAddress string        `yaml:"words_address" env:"WORDS_ADDRESS" env-default:"localhost:81"`
	NatsAddress  string        `yaml:"nats_address" env:"BROKER_ADDRESS" env-default:"nats://nats:4222"`
	DBAddress    string        `yaml:"db_address" env:"DB_ADDRESS" env-default:"localhost:82"`
}

func MustLoad(configPath string) Config {
	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config %q: %s", configPath, err)
	}
	return cfg
}
