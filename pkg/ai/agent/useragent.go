package agent

import (
	"bufio"
	"context"
	"fmt"
	"lfg/pkg/ai"
	"lfg/pkg/ai/agent/planningagent"
	"lfg/pkg/ai/agent/refiningagent"
	"lfg/pkg/exchange"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	log "github.com/sirupsen/logrus"
)

type UserAgent struct {
	Id     string
	Prompt string
	Memory *ai.AgentMemory
	Tasks  []ai.AgentTask
	Client *openai.Client
	logger *log.Entry
}

func NewUserAgent(agentId string, prompt string, exchanges map[string]*exchange.Exchange) (*UserAgent, error) {
	agent := &UserAgent{
		Id:     agentId,
		Prompt: prompt,
		Memory: &ai.AgentMemory{
			Exchanges: exchanges,
			Data:      make(map[string]any),
		},
		logger: log.WithFields(log.Fields{
			"agent": agentId,
		}),
	}

	agent.InitOpenAIClient()

	return agent, nil
}

func (a *UserAgent) InitOpenAIClient() error {
	openaiBaseUrl := os.Getenv("OPENAI_BASE_URL")
	if openaiBaseUrl == "" {
		openaiBaseUrl = "https://openrouter.ai/api/v1/"
	}

	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is not set")
	}

	a.Client = openai.NewClient(
		option.WithBaseURL(openaiBaseUrl),
		option.WithAPIKey(openaiApiKey),
	)

	return nil
}

func (a *UserAgent) Plan(ctx context.Context) error {
	// get available exchanges id
	availableExchangesId := []string{}
	for key := range a.Memory.Exchanges {
		availableExchangesId = append(availableExchangesId, key)
	}

	// init internal agents
	planningAgent := planningagent.NewPlanningAgent(a.Client, availableExchangesId)
	refiningAgent := refiningagent.NewRefiningAgent(a.Client, availableExchangesId)

	// Generate initial plan
	plan, err := planningAgent.Execute(ctx, a.Prompt)
	if err != nil {
		return err
	}
	a.logger.Infof("Initial plan: %v", planningagent.GetReadablePlan(*plan))

	// Wait for user feedback
	reader := bufio.NewReader(os.Stdin)
	log.Println("Enter your comment (leave blank for no comment):")
	userComment, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	// Refine plan
	refinerFeedback, err := refiningAgent.Execute(ctx, a.Prompt, plan, userComment)
	if err != nil {
		return err
	}
	a.logger.Infof("Refiner feedback: %v: %v", refinerFeedback.Type, refinerFeedback.Feedback)

	if refinerFeedback.Type == "FEEDBACK" {
		// Generate new plan based on feedback and user comment
		plan, err = planningAgent.Execute(ctx, refinerFeedback.Feedback)
		if err != nil {
			return err
		}
	} else if refinerFeedback.Type == "NOT_ENOUGH_TOOLS" {
		return fmt.Errorf("not enough tools to execute the plan: %v", refinerFeedback.Feedback)
	}

	// Store tasks and init memory
	a.logger.Infof("Storing tasks...")
	a.logger.Infof("Initiating memory...")
	for _, task := range plan.Tasks {
		task, err := ai.GetTaskByName(task.Name, task.Parameters)
		if err != nil {
			return err
		}
		a.Tasks = append(a.Tasks, *task)
	}

	for _, state := range plan.InitState {
		a.Memory.Set(state.Key, state.Value)
	}

	return nil
}

func (a *UserAgent) Execute(ctx context.Context) error {
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
