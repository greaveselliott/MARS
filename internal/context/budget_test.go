package context

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartition_zeroBudgetReturnsZeroes(t *testing.T) {
	t.Parallel()
	p := Partition(0)
	require.Equal(t, 0, p.SystemTokens)
	require.Equal(t, 0, p.HistoryTokens)
	require.Equal(t, 0, p.ToolTokens)
}

func TestPartition_splitsSensibly(t *testing.T) {
	t.Parallel()
	p := Partition(4000)
	require.Equal(t, 1000, p.SystemTokens)
	require.Equal(t, 600, p.ToolTokens)
	require.Equal(t, 2400, p.HistoryTokens)
	require.Equal(t, 4000, p.SystemTokens+p.HistoryTokens+p.ToolTokens)
}

func TestPartition_smallBudgetNoNegatives(t *testing.T) {
	t.Parallel()
	p := Partition(10)
	require.GreaterOrEqual(t, p.SystemTokens, 0)
	require.GreaterOrEqual(t, p.HistoryTokens, 0)
	require.GreaterOrEqual(t, p.ToolTokens, 0)
	require.Equal(t, 10, p.SystemTokens+p.HistoryTokens+p.ToolTokens)
}
