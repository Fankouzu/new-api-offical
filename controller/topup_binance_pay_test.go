package controller

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBinancePayTradeNoMatchesProviderFormat(t *testing.T) {
	tradeNo := newBinancePayTradeNo(123)

	require.Regexp(t, regexp.MustCompile(`^[A-Za-z0-9]+$`), tradeNo)
	require.LessOrEqual(t, len(tradeNo), 32)
	require.Contains(t, tradeNo, "BP123")
}
