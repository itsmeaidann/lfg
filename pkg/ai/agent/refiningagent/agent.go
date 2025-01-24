package refiningagent

import (
	"context"
	"encoding/json"
	"fmt"
	"lfg/pkg/ai"
	"lfg/pkg/ai/agent/planningagent"
	"lfg/pkg/types"

	"github.com/openai/openai-go"
)

type RefiningAgent struct {
	availableExchangesId []string
	client               *openai.Client
	messages             []types.Message
}

func NewRefiningAgent(client *openai.Client, availableExchangesId []string) *RefiningAgent {
	return &RefiningAgent{
		availableExchangesId: availableExchangesId,
		client:               client,
		messages:             []types.Message{},
	}
}

func (a *RefiningAgent) Execute(ctx context.Context, question string, executionPlan *planningagent.ExecutionPlan, userComment string) (*Feedback, error) {
	fmt.Println("Refining execution plan...")
	tasks := ai.GetAllTaskInterfaces()
	refinerPrompt, err := getRefinerPrompt(question, executionPlan, tasks, a.availableExchangesId, userComment)
	if err != nil {
		return nil, err
	}
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(refinerPrompt),
	}
	chatCompletion, err := a.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F(messages),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(feedbackSchemaParam),
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

	var feedback Feedback
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &feedback)
	if err != nil {
		return nil, err
	}
	a.messages = append(a.messages, types.Message{
		Role:    "user",
		Content: fmt.Sprintf("User query: %s, Current plan: %v, User comment: %s", question, executionPlan, userComment),
	})
	a.messages = append(a.messages, types.Message{
		Role:    "assistant",
		Content: chatCompletion.Choices[0].Message.Content,
	})
	return &feedback, nil
}
