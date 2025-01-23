package ai

import (
	"context"
	"encoding/json"
	"fmt"
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
	memory.SetAsStr(t.OutputKey, klines)
	return nil
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

// MARK: allTasks
func GetAllTasks() []AgentTask {
	return []AgentTask{
		{
			BaseTask: BaseTask{
				Name:        "getKline",
				Description: "Get the kline of the asset using symbol from symbolKey and store the kline in outputKey",
				Parameters: map[string]string{
					"exchangeIdKey": "the key of the exchange id value in the memory that is given",
					"symbolKey":     "the key of the symbol value in the memory (format 'TICKER_USD' not 'TICKER_USDT')",
					"intervalKey":   "the key of the interval value in the memory ex. 'interval1' : '15m'",
					"windowKey":     "the key of the window value in the memory ex. 'window1' : '20'",
					"outputKey":     "the key of the output value in the memory ex. 'kline1' : '10000'",
				},
			},
			Executable: &GetKlineTask{},
		},
	}
}
