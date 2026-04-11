package context

// BudgetPartition splits a total token budget across the three main consumers (M1.4.4).
// Ratios are tuneable; these defaults reserve most space for conversation history
// since tool results are truncated per-call by the tool system.
type BudgetPartition struct {
	SystemTokens  int
	HistoryTokens int
	ToolTokens    int
}

// Partition divides totalBudget into system prompt, conversation history, and tool result reserves.
// Zero totalBudget returns all zeroes (unlimited mode).
func Partition(totalBudget int) BudgetPartition {
	if totalBudget <= 0 {
		return BudgetPartition{}
	}
	sys := totalBudget * 25 / 100
	tool := totalBudget * 15 / 100
	hist := totalBudget - sys - tool
	return BudgetPartition{
		SystemTokens:  sys,
		HistoryTokens: hist,
		ToolTokens:    tool,
	}
}
