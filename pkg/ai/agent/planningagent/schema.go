package planningagent

import (
	"fmt"
	"lfg/pkg/utils"

	"github.com/openai/openai-go"
)

type ExecutionPlan struct {
	Reasoning string       `json:"Reasoning" jsonschema_description:"The brief and clear reasoning of your logic. How does the tasks works and how does the parameters works after each tool execution."`
	Tasks     []TaskFromAI `json:"Tasks" jsonschema_description:"The tasks to be executed periodically for the trading strategy"`
	InitState []Memory     `json:"InitState" jsonschema_description:"The initial state of the trading strategy"`
}

type Memory struct {
	Key   string `json:"Key" jsonschema_description:"The key of the memory"`
	Value string `json:"Value" jsonschema_description:"The value of the memory"`
}

type TaskFromAI struct {
	Name       string            `json:"Name" jsonschema_description:"The exact function name of the tool to be executed"`
	Parameters map[string]string `json:"Parameters" jsonschema_description:"The parameters of the tool"`
}

var ExecutionPlanSchema, _ = utils.GenerateSchema[ExecutionPlan]()

var planSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name:        openai.F("ExecutionPlan"),
	Description: openai.F("The execution plan for the trading strategy to the user query"),
	Schema:      openai.F(ExecutionPlanSchema),
}

func GetReadablePlan(plan ExecutionPlan) string {
	str := fmt.Sprintf("Reasoning: %s\n\nTasks:", plan.Reasoning)
	for _, task := range plan.Tasks {
		str += fmt.Sprintf("\n\t- %s", task.Name)
		for key, value := range task.Parameters {
			str += fmt.Sprintf("\n\t\t- %s: %s", key, value)
		}
	}

	str += "\nInitState:"
	for _, state := range plan.InitState {
		str += fmt.Sprintf("\n- %s: %s", state.Key, state.Value)
	}
	str += "\n\n"
	return str
}
