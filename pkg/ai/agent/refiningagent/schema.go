package refiningagent

import (
	"lfg/pkg/utils"

	"github.com/openai/openai-go"
)

type Feedback struct {
	Feedback string `json:"Feedback" jsonschema_description:"The feedback of the execution plan"`
	Type     string `json:"Type" enum:"CORRECT,NOT_ENOUGH_TOOLS,FEEDBACK"`
}

var FeedbackSchema, _ = utils.GenerateSchema[Feedback]()

var feedbackSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
	Name:        openai.F("Feedback"),
	Description: openai.F("The feedback of the execution plan"),
	Schema:      openai.F(FeedbackSchema),
}
