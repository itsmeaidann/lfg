package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"lfg/pkg/indicator"
	"strings"
)

type BaseTask struct {
	Name        string
	Description string
	Parameters  map[string]string
}

type Executable interface {
	Execute(ctx context.Context, memory *AgentMemory) error
}

type AgentTask struct {
	BaseTask
	Executable Executable
}

func GetAllTaskInterfaces() []BaseTask {
	allTasks := GetAllTasks()
	allTaskInterfaces := []BaseTask{}
	for _, task := range allTasks {
		allTaskInterfaces = append(allTaskInterfaces, task.BaseTask)
	}
	return allTaskInterfaces
}

func GetTaskByName(name string, params map[string]string) (*AgentTask, error) {
	for _, task := range GetAllTasks() {
		if task.Name == name {
			res := task
			// Convert params map to JSON
			paramsJSON, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal params: %w", err)
			}
			if err := json.Unmarshal(paramsJSON, &res.Executable); err != nil {
				return nil, fmt.Errorf("failed to unmarshal params: %w", err)
			}
			return &res, nil
		}
	}
	return nil, nil
}

// MARK: GetKlineTask
type GetKlineTask struct {
	ExchangeIdKey string `json:"exchangeIdKey"`
	IntervalKey   string `json:"intervalKey"`
	SymbolKey     string `json:"symbolKey"`
	WindowKey     string `json:"windowKey"`
	OutputKey     string `json:"outputKey"`
}

func (t *GetKlineTask) Execute(ctx context.Context, memory *AgentMemory) error {

	// parse params
	symbol, err := memory.GetAsStr(t.SymbolKey)
	if err != nil {
		return err
	}
	interval, err := memory.GetAsInterval(t.IntervalKey)
	if err != nil {
		return err
	}
	window, err := memory.GetAsInt(t.WindowKey)
	if err != nil {
		return err
	}
	exchangeId, err := memory.GetAsStr(t.ExchangeIdKey)
	if err != nil {
		return err
	}

	klines, err := (*memory.Exchanges[exchangeId]).GetKLines(symbol, interval, window)
	if err != nil {
		return err
	}

	// write result to memory
	memory.SetAsKlines(t.OutputKey, klines)
	return nil
}

// MARK: GetMovingAverageTask
type GetMovingAverageTask struct {
	KlineKey  string `json:"klineKey"`
	WindowKey string `json:"windowKey"`
	OutputKey string `json:"outputKey"`
}

func (t *GetMovingAverageTask) Execute(ctx context.Context, memory *AgentMemory) error {

	klines, err := memory.GetAsKlines(t.KlineKey)
	if err != nil {
		return err
	}
	// calculate moving average
	window, err := memory.GetAsInt(t.WindowKey)
	if err != nil {
		return err
	}

	maValues := indicator.CalculateMovingAverage(klines, window)
	// write result to memory
	memory.SetAsStr(t.OutputKey, maValues)
	return nil
}

// MARK: AskAITask
type AskAITask struct {
	PromptKey string `json:"promptKey"`
	DataKeys  string `json:"dataKeys"`
	OutputKey string `json:"outputKey"`
}

func (t *AskAITask) Execute(ctx context.Context, memory *AgentMemory) error {
	prompt, err := memory.GetAsStr(t.PromptKey)
	if err != nil {
		return err
	}

	prompt += `\n\nAVAILABLE DATA:\n`
	dataKeys := strings.Split(t.DataKeys, ",")
	for _, dataKey := range dataKeys {
		data, err := memory.GetAsStr(dataKey)
		if err != nil {
			return err
		}
		prompt += fmt.Sprintf("- %s: %s\n", dataKey, data)
	}

	aiResponse, err := GetCompletion(ctx, OpenAIClient, prompt)
	if err != nil {
		return err
	}

	fmt.Println("aiResponse: ", aiResponse)

	memory.SetAsStr(t.OutputKey, aiResponse)
	return nil
}

// MARK: allTasks
func GetAllTasks() []AgentTask {
	return []AgentTask{
		{
			BaseTask: BaseTask{
				Name:        "getKline",
				Description: "Get the kline of the asset using symbol from symbolKey and store the kline in outputKey",
				Parameters: map[string]string{
					"exchangeIdKey": "the key of the exchange id value in the memory that is set by the agent",
					"symbolKey":     "the key of the symbol value in the memory (format 'TICKER_USD' not 'TICKER_USDT')",
					"intervalKey":   "the key of the interval value in the memory ex. 'interval1' : '15m'",
					"windowKey":     "the key of the window value in the memory ex. 'window1' : '20'",
					"outputKey":     "the key of the output value in the memory ex. 'kline1' : '10000'",
				},
			},
			Executable: &GetKlineTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "getMovingAverage",
				Description: "Get the moving average of the asset using kline from klineKey and store the moving average in outputKey. Note that the kline is a list of kline events and if the window is equal to the length of the kline, the moving average will only have last value",
				Parameters: map[string]string{
					"klineKey":  "the key of the kline value in the memory",
					"windowKey": "the key of the window value in the memory ex. 'window1' : '20'",
					"outputKey": "the key of the output value in the memory as []float64",
				},
			},
			Executable: &GetMovingAverageTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "askAI",
				Description: "Ask the AI to answer a question along with the available data in dataKeys and store the answer in outputKey",
				Parameters: map[string]string{
					"promptKey": "the key of the prompt to be asked to the AI in the memory. be specific and clear",
					"dataKeys":  "the keys of the data values in the memory separated by comma ex. 'data1,data2,data3'",
					"outputKey": "the key of the output value in the memory",
				},
			},
			Executable: &AskAITask{},
		},
	}
}
