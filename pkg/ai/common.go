package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"lfg/pkg/utils"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var SharedOpenAIClient *openai.Client

func init() {
	openaiBaseUrl := os.Getenv("OPENAI_BASE_URL")
	if openaiBaseUrl == "" {
		openaiBaseUrl = "https://openrouter.ai/api/v1/"
	}

	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		panic("OPENAI_API_KEY is not set")
	}

	SharedOpenAIClient = openai.NewClient(
		option.WithBaseURL(openaiBaseUrl),
		option.WithAPIKey(openaiApiKey),
	)
}

func GetCompletion(ctx context.Context, client *openai.Client, prompt string) (string, error) {

	chatCompletion, err := client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			Model:       openai.F("openai/gpt-4o-mini"),
			Temperature: openai.F(0.23),
		},
	)
	if err != nil {
		return "", err
	}
	if len(chatCompletion.Choices) == 0 {
		return "", fmt.Errorf("no completion found")
	}
	return chatCompletion.Choices[0].Message.Content, nil
}

func GetStructuredCompletion(ctx context.Context, client *openai.Client, prompt string) (map[string]string, error) {

	var schema, _ = utils.GenerateSchema[map[string]string]()
	chatCompletion, err := client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(
						openai.ResponseFormatJSONSchemaJSONSchemaParam{
							Name:        openai.F("JSON Response"),
							Description: openai.F("The JSON response from the AI"),
							Schema:      openai.F(schema),
						},
					),
				},
			),
			Model:       openai.F("openai/gpt-4o-mini"),
			Temperature: openai.F(0.23),
		},
	)
	if err != nil {
		return map[string]string{}, err
	}
	if len(chatCompletion.Choices) == 0 {
		return map[string]string{}, fmt.Errorf("no completion found")
	}
	var jsonRes map[string]string

	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &jsonRes)
	if err != nil {
		return map[string]string{}, err
	}
	return jsonRes, nil
}
