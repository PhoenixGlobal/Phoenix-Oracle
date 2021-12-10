package test

import (
	"PhoenixOracle/gophoenix/core/store/models"
	"gopkg.in/guregu/null.v3"
	"testing"

"PhoenixOracle/gophoenix/core/adapters"
"github.com/stretchr/testify/assert"
)

func TestEthereumBytes32Formatting(t *testing.T) {
	tests := []struct {
		value    null.String
		expected string
	}{
		{null.StringFrom("16800.00"), "31363830302e3030000000000000000000000000000000000000000000000000"},
		{null.StringFrom(""), "0000000000000000000000000000000000000000000000000000000000000000"},
		{null.StringFrom("Hello World!"), "48656c6c6f20576f726c64210000000000000000000000000000000000000000"},
		{null.StringFromPtr(nil),"0000000000000000000000000000000000000000000000000000000000000000",
		},
	}

	for _, test := range tests {
		past := models.RunResult{
			Output: models.Output{"value": test.value},
		}
		adapter := adapters.EthBytes32{}
		result := adapter.Perform(past,nil)

		assert.Equal(t, test.expected, result.Value())
		assert.Nil(t, result.GetError())
	}
}

