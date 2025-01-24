package utils

import "github.com/invopop/jsonschema"

func GenerateSchema[T any]() (interface{}, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema, nil
}
