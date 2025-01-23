package types

type EnvName string
type YamlMode string

const (
	EnvProd  = EnvName("prod")
	EnvDev   = EnvName("dev")
	EnvLocal = EnvName("local")
)

const (
	YamlModeLocal = YamlMode("LOCAL")
	YamlModeS3    = YamlMode("S3")
)
