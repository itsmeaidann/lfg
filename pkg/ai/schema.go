package ai

import (
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
)

type ExecutionPlan struct {
	Reasoning string       `json:"Reasoning" jsonschema_description:"The brief and clear reasoning of your logic. How does the tasks works and how does the parameters works after each tool execution."`
	Tasks     []TaskFromAI `json:"Tasks" jsonschema_description:"The tasks to be executed periodically for the trading strategy"`
	InitState []Memory     `json:"InitState" jsonschema_description:"The initial state of the trading strategy"`
}

type Feedback struct {
	Feedback string `json:"Feedback" jsonschema_description:"The feedback of the execution plan"`
	Type     string `json:"Type" enum:"CORRECT,NOT_ENOUGH_TOOLS,FEEDBACK"`
}

type Memory struct {
	Key   string `json:"Key" jsonschema_description:"The key of the memory"`
	Value string `json:"Value" jsonschema_description:"The value of the memory"`
}

type TaskFromAI struct {
	Name       string            `json:"Name" jsonschema_description:"The exact function name of the tool to be executed"`
	Parameters map[string]string `json:"Parameters" jsonschema_description:"The parameters of the tool"`
}

func GenerateSchema[T any]() (interface{}, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema, nil
}

var ExecutionPlanSchema, _ = GenerateSchema[ExecutionPlan]()
var FeedbackSchema, _ = GenerateSchema[Feedback]()

var planSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name:        openai.F("ExecutionPlan"),
	Description: openai.F("The execution plan for the trading strategy to the user query"),
	Schema:      openai.F(ExecutionPlanSchema),
}

var feedbackSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name:        openai.F("Feedback"),
	Description: openai.F("The feedback of the execution plan"),
	Schema:      openai.F(FeedbackSchema),
}
