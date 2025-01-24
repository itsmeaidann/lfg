package refiningagent

import (
	"encoding/json"
	"fmt"
	"lfg/pkg/ai"
	"lfg/pkg/ai/agent/planningagent"
)

const RefinerPrompt = `
You are a trading strategy execution plan validator. Your task is to rigorously verify and improve the proposed execution plan.

AVAILABLE TOOLS:
<available_tools>
%s
</available_tools>

CURRENT EXECUTION PLAN:
<current_tasks>
%s
</current_tasks>

USER STRATEGY: "%s"

AVAILABLE EXCHANGES:
<available_exchanges_id>
%s
</available_exchanges_id>

VALIDATION CHECKLIST:
1. Tool Availability
   - All specified tools exist in AVAILABLE_TOOLS
   - Tool parameters match their definitions

2. Data Flow Validation
   - All input keys are produced by previous steps or initial state
   - No circular dependencies
   - Data types are consistent

3. Strategy Requirements
   - Plan fully implements the user's strategy
   - No missing steps or edge cases
   - Correct order of operations

4. Exchange Compatibility
   - Symbol format is "TICKER_USD"
   - Exchange IDs are valid
   - Operations are supported by chosen exchange

RESPONSE FORMAT:
{
    "type": "CORRECT" | "FEEDBACK" | "NOT_ENOUGH_TOOLS",
    "details": "Detailed explanation of issues found or confirmation of correctness",
    "suggestions": [
        "Specific improvements or fixes needed"
    ]
}

USER LATEST COMMENT: "%s"

IMPORTANT:
- Be precise and specific in your feedback
- Only report actual issues that affect execution
- Verify each step's logic and data flow
- Consider edge cases and error scenarios
`

// get system prompt for refining execution plan
func getRefinerPrompt(question string, executionPlan *planningagent.ExecutionPlan, tasks []ai.BaseTask, availableExchangesId []string, userComment string) (string, error) {
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

	refinerPrompt := fmt.Sprintf(RefinerPrompt, tasksDescription, string(executionPlanJson), question, availableExchangesId, userComment)
	return refinerPrompt, nil
}
