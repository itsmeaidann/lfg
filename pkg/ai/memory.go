package ai

import (
	"fmt"
	"lfg/pkg/exchange"
	"lfg/pkg/types"
	"strconv"
)

type AgentMemory struct {
	Exchanges map[string]*exchange.Exchange
	Data      map[string]any
}

type MemoryType string

const (
	MemoryTypeStr      = MemoryType("string")
	MemoryTypeFloat64  = MemoryType("float64")
	MemoryTypeInt64    = MemoryType("int64")
	MemoryTypeInterval = MemoryType("interval")
)

// MARK: setters

// store memory value
func (m *AgentMemory) Set(key string, value any) {
	m.Data[key] = value
}

// store memory value
func (m *AgentMemory) SetAsStr(key string, value any) {
	m.Data[key] = fmt.Sprintf("%v", value)
}

// MARK: getters

// retrieve memory value
func (m *AgentMemory) Get(key string) any {
	return m.Data[key]
}

// retrieve memory value as a string
func (m *AgentMemory) GetAsStr(key string) (string, error) {
	value, ok := m.Data[key]
	if !ok {
		return "", fmt.Errorf("key '%v' not found in agent memory", key)
	}

	casted, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value of key '%v' not stored as string", key)
	}

	return casted, nil
}

// retrieve memory value as a float64
func (m *AgentMemory) GetAsFloat64(key string) (float64, error) {
	strValue, err := m.GetAsStr(key)
	if err != nil {
		return 0, err
	}

	casted, err := strconv.ParseFloat(strValue, 64)
	if err != nil {
		return 0, err
	}
	return casted, nil
}

// retrieve memory value as an int64
func (m *AgentMemory) GetAsInt(key string) (int, error) {
	strValue, err := m.GetAsStr(key)
	if err != nil {
		return 0, err
	}

	casted, err := strconv.Atoi(strValue)
	if err != nil {
		return 0, err
	}
	return casted, nil
}

// retrieve memory value as an int64
func (m *AgentMemory) GetAsInt64(key string) (int64, error) {
	strValue, err := m.GetAsStr(key)
	if err != nil {
		return 0, err
	}

	casted, err := strconv.Atoi(strValue)
	if err != nil {
		return 0, err
	}
	return int64(casted), nil
}

// retrieve memory value as an interval
func (m *AgentMemory) GetAsInterval(key string) (types.Interval, error) {
	strValue, err := m.GetAsStr(key)
	if err != nil {
		return "", err
	}

	casted := types.Interval(strValue)
	return casted, nil
}
