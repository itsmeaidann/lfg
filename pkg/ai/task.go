package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"lfg/pkg/indicator"
	"lfg/pkg/types"
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
	OutputKey string `json:"outputKey"`
}

func (t *GetMovingAverageTask) Execute(ctx context.Context, memory *AgentMemory) error {

	klines, err := memory.GetAsKlines(t.KlineKey)
	if err != nil {
		return err
	}

	maValues := indicator.CalculateMovingAverage(klines, int(len(klines)/2))
	// write result to memory
	memory.SetAsStr(t.OutputKey, maValues)
	return nil
}

// MARK: GetBollingerBandTask
type GetBollingerBandTask struct {
	KlineKey  string `json:"klineKey"`
	OutputKey string `json:"outputKey"`
}

func (t *GetBollingerBandTask) Execute(ctx context.Context, memory *AgentMemory) error {

	klines, err := memory.GetAsKlines(t.KlineKey)
	if err != nil {
		return err
	}

	bollingerBand, err := indicator.CalculateBollingerBand(klines, int(len(klines)/2), 2)
	if err != nil {
		return err
	}
	memory.SetAsStr(t.OutputKey, bollingerBand)
	return nil
}

// MARK: AskAITask
type AskAITask struct {
	Prompt    string `json:"prompt"`
	DataKeys  string `json:"dataKeys"`
	OutputKey string `json:"outputKey"`
}

func (t *AskAITask) Execute(ctx context.Context, memory *AgentMemory) error {
	prompt := `\n\nAVAILABLE DATA:\n`

	dataKeys := strings.Split(t.DataKeys, ",")
	for _, dataKey := range dataKeys {
		data, err := memory.GetAsStr(dataKey)
		if err != nil {
			return err
		}
		prompt += fmt.Sprintf("- %s: %s\n", dataKey, data)
	}
	prompt += "\n\nUSER INSTRUCTION: " + t.Prompt

	prompt += "\nIMPORTANT: YOUR OUTPUT WILL BE USED TO SET AS A STR IN THE MEMORY AND USED FURTHER. FOLLOW FORMAT IN THE INSTRUCTION STRICTLY"

	aiResponse, err := GetCompletion(ctx, SharedOpenAIClient, prompt)
	if err != nil {
		return err
	}

	fmt.Println("aiResponse: ", aiResponse)

	memory.SetAsStr(t.OutputKey, aiResponse)
	return nil
}

// MARK: AISetMemoryTask
type AISetMemoryTask struct {
	Prompt   string `json:"prompt"`
	DataKeys string `json:"dataKeys"`
}

func (t *AISetMemoryTask) Execute(ctx context.Context, memory *AgentMemory) error {

	prompt := `\n\nAVAILABLE DATA:\n`
	dataKeys := strings.Split(t.DataKeys, ",")
	for _, dataKey := range dataKeys {
		data, err := memory.GetAsStr(dataKey)
		if err != nil {
			return err
		}
		prompt += fmt.Sprintf("- %s: %s\n", dataKey, data)
	}
	prompt += "\n\nUSER INSTRUCTION: " + t.Prompt
	prompt += "\nIMPORTANT: YOUR OUTPUT WILL BE USED TO SET AS A JSON IN THE MEMORY AND USED FURTHER. FOLLOW FORMAT IN THE INSTRUCTION STRICTLY"

	aiResponse, err := GetStructuredCompletion(ctx, SharedOpenAIClient, prompt)
	if err != nil {
		return err
	}
	fmt.Println("aiResponse: ", aiResponse)

	for key, value := range aiResponse {
		memory.SetAsStr(key, value)
	}
	return nil
}

// MARK: openMarketLongPositionTask
type OpenMarketLongPositionIfTask struct {
	IfKey         string `json:"ifKey"`
	IfValue       string `json:"ifValue"`
	AmountUsdKey  string `json:"amountUsdKey"`
	SymbolKey     string `json:"symbolKey"`
	ExchangeIdKey string `json:"exchangeIdKey"`
}

func (t *OpenMarketLongPositionIfTask) Execute(ctx context.Context, memory *AgentMemory) error {
	ifValue, err := memory.GetAsStr(t.IfKey)
	if err != nil {
		return err
	}
	if ifValue != t.IfValue {
		return nil
	}

	amountUsd, err := memory.GetAsFloat64(t.AmountUsdKey)
	if err != nil {
		return err
	}
	symbol, err := memory.GetAsStr(t.SymbolKey)
	if err != nil {
		return err
	}
	exchangeId, err := memory.GetAsStr(t.ExchangeIdKey)
	if err != nil {
		return err
	}

	klines, err := (*memory.Exchanges[exchangeId]).GetKLines(symbol, types.Interval1m, 1)
	if err != nil {
		return err
	}
	price := klines[len(klines)-1].Kline.C

	amount := amountUsd / price

	// TODO: make leverage dynamic
	lev := 5
	err = (*memory.Exchanges[exchangeId]).OpenMarketOrder(symbol, types.OrderSideBuy, amount, lev, false)
	if err != nil {
		return err
	}

	return nil
}

// MARK: openMarketShortPositionTask
type OpenMarketShortPositionIfTask struct {
	IfKey         string `json:"ifKey"`
	IfValue       string `json:"ifValue"`
	AmountUsdKey  string `json:"amountUsdKey"`
	SymbolKey     string `json:"symbolKey"`
	ExchangeIdKey string `json:"exchangeIdKey"`
}

func (t *OpenMarketShortPositionIfTask) Execute(ctx context.Context, memory *AgentMemory) error {
	ifValue, err := memory.GetAsStr(t.IfKey)
	if err != nil {
		return err
	}
	if ifValue != t.IfValue {
		return nil
	}

	amountUsd, err := memory.GetAsFloat64(t.AmountUsdKey)
	if err != nil {
		return err
	}
	symbol, err := memory.GetAsStr(t.SymbolKey)
	if err != nil {
		return err
	}
	exchangeId, err := memory.GetAsStr(t.ExchangeIdKey)
	if err != nil {
		return err
	}

	klines, err := (*memory.Exchanges[exchangeId]).GetKLines(symbol, types.Interval1m, 1)
	if err != nil {
		return err
	}
	price := klines[len(klines)-1].Kline.C

	amount := amountUsd / price

	// TODO: make leverage dynamic
	lev := 5
	err = (*memory.Exchanges[exchangeId]).OpenMarketOrder(symbol, types.OrderSideSell, amount, lev, false)
	if err != nil {
		return err
	}

	return nil
}

// MARK: openLimitLongPositionTask
type OpenLimitLongPositionIfTask struct {
	IfKey         string `json:"ifKey"`
	IfValue       string `json:"ifValue"`
	PriceKey      string `json:"priceKey"`
	AmountUsdKey  string `json:"amountUsdKey"`
	SymbolKey     string `json:"symbolKey"`
	ExchangeIdKey string `json:"exchangeIdKey"`
}

func (t *OpenLimitLongPositionIfTask) Execute(ctx context.Context, memory *AgentMemory) error {
	ifValue, err := memory.GetAsStr(t.IfKey)
	if err != nil {
		return err
	}
	if ifValue != t.IfValue {
		return nil
	}

	amountUsd, err := memory.GetAsFloat64(t.AmountUsdKey)
	if err != nil {
		return err
	}
	symbol, err := memory.GetAsStr(t.SymbolKey)
	if err != nil {
		return err
	}
	exchangeId, err := memory.GetAsStr(t.ExchangeIdKey)
	if err != nil {
		return err
	}

	price, err := memory.GetAsFloat64(t.PriceKey)
	if err != nil {
		return err
	}

	amount := amountUsd / price

	// TODO: make leverage dynamic
	lev := 5
	_, err = (*memory.Exchanges[exchangeId]).OpenLimitOrder(symbol, types.OrderSideBuy, price, amount, lev, false, types.OrderTIFGTC, "")
	if err != nil {
		return err
	}

	return nil
}

// MARK: openShortPositionTask
type OpenLimitShortPositionIfTask struct {
	IfKey         string `json:"ifKey"`
	IfValue       string `json:"ifValue"`
	PriceKey      string `json:"priceKey"`
	AmountUsdKey  string `json:"amountUsdKey"`
	SymbolKey     string `json:"symbolKey"`
	ExchangeIdKey string `json:"exchangeIdKey"`
}

func (t *OpenLimitShortPositionIfTask) Execute(ctx context.Context, memory *AgentMemory) error {
	ifValue, err := memory.GetAsStr(t.IfKey)
	if err != nil {
		return err
	}
	if ifValue != t.IfValue {
		return nil
	}

	amountUsd, err := memory.GetAsFloat64(t.AmountUsdKey)
	if err != nil {
		return err
	}
	symbol, err := memory.GetAsStr(t.SymbolKey)
	if err != nil {
		return err
	}
	exchangeId, err := memory.GetAsStr(t.ExchangeIdKey)
	if err != nil {
		return err
	}

	price, err := memory.GetAsFloat64(t.PriceKey)
	if err != nil {
		return err
	}

	amount := amountUsd / price

	// TODO: make leverage dynamic
	lev := 5
	_, err = (*memory.Exchanges[exchangeId]).OpenLimitOrder(symbol, types.OrderSideSell, price, amount, lev, false, types.OrderTIFGTC, "")
	if err != nil {
		return err
	}

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
					"outputKey": "the key of the output value in the memory as []float64",
				},
			},
			Executable: &GetMovingAverageTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "getBollingerBand",
				Description: "Get the bollinger band of the asset using kline from klineKey and store the bollinger band in outputKey",
				Parameters: map[string]string{
					"klineKey":  "the key of the kline value in the memory",
					"outputKey": "the key of the output value in the memory as []float64",
				},
			},
			Executable: &GetBollingerBandTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "openMarketLongPositionIf",
				Description: "Open a market long position of the symbolKey using amountUsd from amountUsdKey IF the value of ifKey in the memory is equal to ifValue",
				Parameters: map[string]string{
					"ifKey":         "the key of the if value in the memory",
					"ifValue":       "the value of the if value in the memory",
					"amountUsdKey":  "the key of the amount usd position size in the memory",
					"symbolKey":     "the key of the symbol value in the memory (format 'TICKER_USD' not 'TICKER_USDT')",
					"exchangeIdKey": "the key of the exchange id value in the memory that is set by the agent",
				},
			},
			Executable: &OpenMarketLongPositionIfTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "openMarketShortPositionIf",
				Description: "Open a market short position of the symbolKey using amountUsd from amountUsdKey IF the value of ifKey in the memory is equal to ifValue",
				Parameters: map[string]string{
					"ifKey":         "the key of the if value in the memory",
					"ifValue":       "the value of the if value in the memory",
					"priceKey":      "the key of the price value in the memory",
					"amountUsdKey":  "the key of the amount usd position size in the memory",
					"symbolKey":     "the key of the symbol value in the memory (format 'TICKER_USD' not 'TICKER_USDT')",
					"exchangeIdKey": "the key of the exchange id value in the memory that is set by the agent",
				},
			},
			Executable: &OpenMarketShortPositionIfTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "openLimitLongPositionIf",
				Description: "Open a limit long position of the symbolKey using amountUsd from amountUsdKey IF the value of ifKey in the memory is equal to ifValue",
				Parameters: map[string]string{
					"ifKey":         "the key of the if value in the memory",
					"ifValue":       "the value of the if value in the memory",
					"priceKey":      "the key of the price value in the memory",
					"amountUsdKey":  "the key of the amount usd position size in the memory",
					"symbolKey":     "the key of the symbol value in the memory (format 'TICKER_USD' not 'TICKER_USDT')",
					"exchangeIdKey": "the key of the exchange id value in the memory that is set by the agent",
				},
			},
			Executable: &OpenLimitLongPositionIfTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "openShortPositionIf",
				Description: "Open a short position of the symbolKey using amountUsd from amountUsdKey IF the value of ifKey in the memory is equal to ifValue",
				Parameters: map[string]string{
					"ifKey":         "the key of the if value in the memory",
					"ifValue":       "the value of the if value in the memory",
					"priceKey":      "the key of the price value in the memory",
					"amountUsdKey":  "the key of the amount usd position size in the memory",
					"symbolKey":     "the key of the symbol value in the memory (format 'TICKER_USD' not 'TICKER_USDT')",
					"exchangeIdKey": "the key of the exchange id value in the memory that is set by the agent",
				},
			},
			Executable: &OpenLimitShortPositionIfTask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "askAI",
				Description: "Ask the AI to answer a question along with the available data in dataKeys and store the answer as string in outputKey",
				Parameters: map[string]string{
					"prompt":    "the prompt to be asked to the AI in the memory. be specific and clear. u MUST CLEARLY outline the output format ex. ONLY OUTPUT 'yes' | 'no' | 'idk'",
					"dataKeys":  "the keys of the data values in the memory separated by comma ex. 'data1,data2,data3'",
					"outputKey": "the key of the output value in the memory",
				},
			},
			Executable: &AskAITask{},
		},
		{
			BaseTask: BaseTask{
				Name:        "aiSetMemory",
				Description: "Ask the AI a query along with the available data in dataKeys and return json that will be set in memory (map[string]string)",
				Parameters: map[string]string{
					"prompt":   "the prompt to be asked to the AI in the memory. be specific and clear. u MUST CLEARLY outline the output format ex. {'name': '...', 'desc': '...'}",
					"dataKeys": "the keys of the data values in the memory separated by comma ex. 'data1,data2,data3'",
				},
			},
			Executable: &AISetMemoryTask{},
		},
	}
}
