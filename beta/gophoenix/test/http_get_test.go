package test

import (
	"PhoenixOracle/gophoenix/core/adapters"
	"PhoenixOracle/gophoenix/core/store/models"
	"github.com/stretchr/testify/assert"
	gock "gopkg.in/h2non/gock.v1"
	"testing"
)

func TestHttpGetNotAUrlError(t *testing.T) {
	t.Parallel()
	httpGet := adapters.HttpGet{Endpoint: "NotAUrl"}
	input := models.RunResult{}
	result := httpGet.Perform(input, nil)
	assert.Nil(t, result.Output)
	assert.NotNil(t, result.Error)
}

func TestHttpGetResponseError(t *testing.T) {
	defer gock.Off()
	url := `https://example.com/api`

	gock.New(url).
		Get("").
		Reply(400).
		JSON(`Invalid request`)

	httpGet := adapters.HttpGet{Endpoint: url}
	input := models.RunResult{}
	result := httpGet.Perform(input, nil)
	assert.Nil(t, result.Output)
	assert.NotNil(t, result.Error)
}
