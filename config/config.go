package config

import (
	"lfg/pkg/types"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Notifications   *NotificationConfig        `yaml:"notifications"`
	Persistence     *PersistenceConfig         `yaml:"persistence"`
	DatabaseConfig  *DatabaseConfig            `yaml:"databaseConfig"`
	ExchangeConfigs map[string]*ExchangeConfig `yaml:"exchange"`
	AgentConfigs    map[string]*AgentConfig    `yaml:"agent"`
}

type NotificationConfig struct {
	// not implemented
}

type PersistenceConfig struct {
	// not implemented
}

type DatabaseConfig struct {
	// not implemented
}

type ExchangeConfig struct {
	ExchangeName types.ExchangeName `yaml:"exchange"`
	EnvPrefix    string             `yaml:"envPrefix"`
	Futures      bool               `yaml:"futures"`
	SubAccountId uint               `yaml:"subAccountId"` // optional
	IsCross      bool               `yaml:"isCross"`
}

type AgentConfig struct {
	Exchange []*string `yaml:"exchange"`
	Prompt   string    `yaml:"prompt"`
}

func LoadConfig(envName types.EnvName) (*Config, error) {
	// read YAML file
	var data []byte
	var err error

	yamlFiles := map[types.EnvName]string{
		types.EnvLocal: "lfg.yaml",
		types.EnvDev:   "lfg.dev.yaml",
		types.EnvProd:  "lfg.prod.yaml",
	}
	fileName := yamlFiles[envName]
	data, err = os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("fail to load config file '%s': %v", fileName, err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("fail to decode config file '%v': %v", config, err)
	}
	return &config, nil
}
