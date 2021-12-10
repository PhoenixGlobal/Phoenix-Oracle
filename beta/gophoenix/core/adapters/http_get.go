package adapters

import (
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HttpGet struct {
	Endpoint string `json:"endpoint"`
}

func (self HttpGet) Perform(input models.RunResult, _ *store.Store) models.RunResult {
	response, err := http.Get(self.Endpoint)
	fmt.Println("***********************")
	fmt.Println(response)
	fmt.Println(self.Endpoint)
	fmt.Println("***********************")
	if err != nil{
		return models.RunResultWithError(err)
	}
	defer response.Body.Close()
	bytes, err:= ioutil.ReadAll(response.Body)
	body := string(bytes)
	if err != nil{
		return models.RunResultWithError(err)
	}
	if response.StatusCode >= 300{
		return models.RunResultWithError(fmt.Errorf(body))
	}
	fmt.Println("!!!!!!!!!!!!!!!!!!!")
	rs :=  models.RunResultWithValue(body)
	fmt.Println(rs)
	fmt.Println("!!!!!!!!!!!!!!!!!!!!")
	return rs
}

