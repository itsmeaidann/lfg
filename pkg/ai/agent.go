package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"lfg/pkg/exchange"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	log "github.com/sirupsen/logrus"
)

type Agent struct {
	Id     string
	Prompt string
	Memory *AgentMemory
	Tasks  []AgentTask

	logger *log.Entry
}

func NewAgent(agentId string, prompt string, exchanges map[string]*exchange.Exchange) (*Agent, error) {
	agent := &Agent{
		Id:     agentId,
		Prompt: prompt,
		Memory: &AgentMemory{
			Exchanges: exchanges,
			Data:      make(map[string]any),
		},
		logger: log.WithFields(log.Fields{
			"agent": agentId,
		}),
	}
	return agent, nil
}

func (a *Agent) Plan(ctx context.Context) error {
	// setup openai client
	openaiBaseUrl := os.Getenv("OPENAI_BASE_URL")
	if openaiBaseUrl == "" {
		openaiBaseUrl = "https://openrouter.ai/api/v1/"
	}

	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is not set")
	}

	client := openai.NewClient(
		option.WithBaseURL(openaiBaseUrl),
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)

	// setup variables
	refined := false
	prevMessages := []openai.ChatCompletionMessageParamUnion{}
	refineCount := 0
	maxRefineCount := 5

	// get available exchanges id
	availableExchangesId := []string{}
	for key := range a.Memory.Exchanges {
		availableExchangesId = append(availableExchangesId, key)
	}

	// generate execution plan
	a.logger.Println("Starting planning...")
	var plan ExecutionPlan
	var refinedFeedback Feedback

	// refine execution plan until it is correct or max refine count is reached
	for !refined && refineCount < maxRefineCount {
		var err error
		// generate execution plan
		plan, err = GenerateExecutionPlan(ctx, client, availableExchangesId, a.Prompt, prevMessages)
		if err != nil {
			return err
		}
		jsonPlan, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		// log initial reasoning and plan
		a.logger.Infof("Initial Reasoning #%v: \n%v\n", refineCount+1, plan.Reasoning)
		a.logger.Debugf("Initial Plan #%v: \n%v\n", refineCount+1, string(jsonPlan))

		// refine execution plan
		refinedFeedback, err = RefineExecutionPlan(ctx, client, availableExchangesId, a.Prompt, plan)
		if err != nil {
			return err
		}

		a.logger.Infof("Refiner Feedback #%v: \n%v\n", refineCount+1, refinedFeedback.Feedback)

		// check if the refined feedback is correct or not
		if refinedFeedback.Type == "NOT_ENOUGH_TOOLS" {
			refined = false
			a.logger.Warn("NOT_ENOUGH_TOOLS")
			break
		} else if refinedFeedback.Type == "CORRECT" {
			refined = true
			a.logger.Infof("Refined Successfully...")
		} else {
			refined = false
			prevMessages = append(prevMessages, openai.AssistantMessage(string(jsonPlan)))
			prevMessages = append(prevMessages, openai.UserMessage(refinedFeedback.Feedback))
			refineCount++
		}
	}

	// if refined successfully, store the tasks and initiate memory
	if refined {
		a.logger.Infof("Storing tasks...")
		a.logger.Debugf("Tasks: %v", plan.Tasks)
		a.logger.Infof("Initiating memory...")
		a.logger.Debugf("InitState: %v", plan.InitState)
		for _, task := range plan.Tasks {
			task, err := GetTaskByName(task.Name, task.Parameters)
			if err != nil {
				return err
			}
			a.Tasks = append(a.Tasks, *task)
		}

		for _, state := range plan.InitState {
			a.Memory.Set(state.Key, state.Value)
		}

	} else {
		return fmt.Errorf("refined failed, skipping memory initiation:[%v] %v", refinedFeedback.Type, refinedFeedback.Feedback)
	}
	return nil
}

func (a *Agent) Execute(ctx context.Context) error {
	// execute all tasks
	for _, task := range a.Tasks {
		a.logger.Infof("Executing task: %v", task.Name)
		err := task.Executable.Execute(ctx, a.Memory)
		if err != nil {
			a.logger.Errorf("Error executing task %v: %v", task.Name, err)
		}
		a.logger.Infof("Task %v executed successfully", task.Name)
		a.logger.Infof("Current Memory: %v", a.Memory.Data)
	}

	return nil
}
