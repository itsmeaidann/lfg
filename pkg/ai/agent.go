package ai

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"lfg/pkg/exchange"

	"github.com/openai/openai-go"
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
	err := InitOpenAIClient()
	if err != nil {
		return nil, err
	}

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
	// setup variables
	refined := false
	prevMessages := []openai.ChatCompletionMessageParamUnion{}
	refineCount := 0
	maxRefineCount := 3

	// get available exchanges id
	availableExchangesId := []string{}
	for key := range a.Memory.Exchanges {
		availableExchangesId = append(availableExchangesId, key)
	}

	// generate execution plan
	a.logger.Println("Starting planning...")
	var plan ExecutionPlan
	var refinedFeedback Feedback
	var userNoComment bool = false

	// refine execution plan until it is correct or max refine count is reached
	for !refined && refineCount <= maxRefineCount {
		var err error
		// generate execution plan
		plan, err = GenerateExecutionPlan(ctx, OpenAIClient, availableExchangesId, a.Prompt, prevMessages)
		if err != nil {
			return err
		}
		jsonPlan, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return err
		}
		// log initial reasoning and plan
		a.logger.Infof("Round %v: \n%v\n", refineCount+1, GetReadablePlan(plan))

		// wait for user comment through stdin
		userComment := "NO COMMENT"
		if !userNoComment {
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Enter your comment (leave blank for no comment):")
			userComment, err = reader.ReadString('\n')
			if err != nil {
				return err
			}
			if userComment == "" {
				userNoComment = true
			}
		}

		// refine execution plan
		refinedFeedback, err = RefineExecutionPlan(ctx, OpenAIClient, availableExchangesId, a.Prompt, plan, userComment)
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
		a.logger.Infof("Initiating memory...")
		a.logger.Infof("Plan: \n%v\n", GetReadablePlan(plan))
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
		return fmt.Errorf("max refine count reached, skipping memory initiation:[%v] %v", refinedFeedback.Type, refinedFeedback.Feedback)
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
	}

	// last memory update log
	a.logger.Infof("final memory state: %v", a.Memory.Data)
	return nil
}
