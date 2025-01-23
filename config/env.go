package config

import (
	"lfg/pkg/types"
	"lfg/pkg/utils"

	"strings"

	"github.com/joho/godotenv"
)

var Env = Environment{}

type Environment struct {
	EnvName types.EnvName
}

func init() {
	godotenv.Load()
	switch env := strings.ToLower(utils.LoadEnv("ENVIRONMENT")); env {
	case "prod", "production":
		Env.EnvName = types.EnvProd
	case "dev", "staging":
		Env.EnvName = types.EnvDev
	default:
		Env.EnvName = types.EnvLocal
	}
}
