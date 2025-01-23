package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
)

// get system prompt for generating execution plan
func getSystemPrompt(query string, prevMessages []openai.ChatCompletionMessageParamUnion, tasks []BaseTask, availableExchangesId []string) (string, error) {
	tasksDescription := ""
	for _, task := range tasks {
		tasksDescription += fmt.Sprintf("- %s\n\tDescription: %s\n\tParameters:\n", task.Name, task.Description)
		for key, value := range task.Parameters {
			tasksDescription += fmt.Sprintf("\t\t- %s (%s)\n", key, value)
		}
	}

	systemPrompt := fmt.Sprintf(SystemPrompt, tasksDescription, prevMessages, query, availableExchangesId)
	return systemPrompt, nil
}

// get system prompt for refining execution plan
func getRefinerPrompt(question string, executionPlan ExecutionPlan, tasks []BaseTask, availableExchangesId []string) (string, error) {
	tasksDescription := ""
	for _, task := range tasks {
		tasksDescription += fmt.Sprintf("- %s\n\tDescription: %s\n\tParameters:\n", task.Name, task.Description)
		for key, value := range task.Parameters {
			tasksDescription += fmt.Sprintf("\t\t- %s (%s)\n", key, value)
		}
	}

	executionPlanJson, err := json.MarshalIndent(executionPlan, "", "  ")
	if err != nil {
		return "", err
	}

	refinerPrompt := fmt.Sprintf(RefinerPrompt, tasksDescription, executionPlanJson, question, availableExchangesId)
	return refinerPrompt, nil
}

// generate execution plan
func GenerateExecutionPlan(ctx context.Context, client *openai.Client, availableExchangesId []string, question string, prevMessages []openai.ChatCompletionMessageParamUnion) (ExecutionPlan, error) {
	fmt.Println("Generating execution plan...")
	fmt.Println(question)
	tasks := GetAllTaskInterfaces()
	systemPrompt, err := getSystemPrompt(question, prevMessages, tasks, availableExchangesId)
	if err != nil {
		return ExecutionPlan{}, err
	}
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}
	chatCompletion, err := client.Chat.Completions.New(
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
		return ExecutionPlan{}, err
	}

	var executionPlan ExecutionPlan
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &executionPlan)
	if err != nil {
		return ExecutionPlan{}, err
	}
	return executionPlan, nil
}

// refine execution plan
func RefineExecutionPlan(ctx context.Context, client *openai.Client, availableExchangesId []string, question string, executionPlan ExecutionPlan) (Feedback, error) {
	fmt.Println("Refining execution plan...")
	tasks := GetAllTaskInterfaces()
	refinerPrompt, err := getRefinerPrompt(question, executionPlan, tasks, availableExchangesId)
	if err != nil {
		return Feedback{}, err
	}
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(refinerPrompt),
	}
	chatCompletion, err := client.Chat.Completions.New(
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
		return Feedback{}, err
	}

	var feedback Feedback
	err = json.Unmarshal([]byte(chatCompletion.Choices[0].Message.Content), &feedback)
	if err != nil {
		return Feedback{}, err
	}
	return feedback, nil
}
