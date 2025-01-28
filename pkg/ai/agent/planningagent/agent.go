package planningagent

import (
	"context"
	"encoding/json"
	"lfg/pkg/ai"
	"lfg/pkg/types"

	log "github.com/sirupsen/logrus"

	"github.com/openai/openai-go"
)

type PlanningAgent struct {
	client               *openai.Client
	availableExchangesId []string
	messages             []types.Message
}

func NewPlanningAgent(client *openai.Client, availableExchangesId []string) *PlanningAgent {
	return &PlanningAgent{
		availableExchangesId: availableExchangesId,
		client:               client,
		messages:             []types.Message{},
	}
}

func (a *PlanningAgent) Execute(ctx context.Context, question string) (*ExecutionPlan, error) {
	log.Println("Generating execution plan...")
	tasks := ai.GetAllTaskInterfaces()
	systemPrompt, err := getSystemPrompt(question, a.messages, tasks, a.availableExchangesId)
	if err != nil {
		return nil, err
	}
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(question),
	}
	chatCompletion, err := a.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F(messages),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(planSchemaParam),
				},
			),
			Model:       openai.F("openai/gpt-4o-mini"),
			Modalities:  openai.F([]openai.ChatCompletionModality{openai.ChatCompletionModality(openai.ChatCompletionNewParamsResponseFormatTypeJSONSchema)}),
			Temperature: openai.F(0.23),
		},
	)
	if err != nil {
		return nil, err
	}

	var executionPlan ExecutionPlan
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &executionPlan)
	if err != nil {
		return nil, err
	}
	a.messages = append(a.messages, types.Message{
		Role:    "user",
		Content: question,
	})
	a.messages = append(a.messages, types.Message{
		Role:    "assistant",
		Content: chatCompletion.Choices[0].Message.Content,
	})

	return &executionPlan, nil
}
