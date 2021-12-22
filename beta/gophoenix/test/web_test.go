package test

import (
	"PhoenixOracle/gophoenix/core/adapters"
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"PhoenixOracle/gophoenix/core/utils"
	"PhoenixOracle/gophoenix/core/web/controllers"
	"bytes"
	"encoding/json"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
	"io/ioutil"
	"net/http"
	"testing"
)


func TestCreateTasks(t *testing.T) {
	t.Parallel()
	app := NewApplication()
	server := app.NewServer()
	defer app.Stop()

	jsonStr := LoadJSON("./fixture/job_integration.json")
	resp, err := BasicAuthPost(server.URL+"/v2/jobs", "application/json", bytes.NewBuffer(jsonStr))
	//if err != nil {
	//	t.Fatal(err)
	//}
	assert.Equal(t, 200, resp.StatusCode, "Response should be success")

	defer resp.Body.Close()
	respJSON := JobJSONFromResponse(resp.Body)
	var j models.Job
	app.Store.One("ID", respJSON.ID, &j)
	assert.Equal(t, j.ID, respJSON.ID, "Wrong job returned")

	adapter1,_ := adapters.For(j.Tasks[0])
	httpGet := adapter1.(*adapters.HttpGet)
	assert.Nil(t, err)
	assert.Equal(t, httpGet.Endpoint, "https://bitstamp.net/api/ticker/")


	adapter2,_ := adapters.For(j.Tasks[1])
	jsonParse := adapter2.(*adapters.JsonParse)
	assert.Equal(t, jsonParse.Path, []string{"last"})

	adapter3,_ := adapters.For(j.Tasks[3])
	signTx := adapter3.(*adapters.EthTx)
	assert.Equal(t, signTx.Address, "0x356a04bce728ba4c62a30294a55e6a8600a320b3")
	assert.Equal(t, signTx.FunctionID, "12345679")

	var initr models.Initiator
	app.Store.One("JobID", j.ID, &initr)
	assert.Equal(t, "web", initr.Type)
}

func TestCreateJobSchedulerIntegration(t *testing.T) {
	RegisterTestingT(t)

	app := NewApplication()
	server := app.NewServer()
	app.Start()
	defer app.Stop()

	jsonStr := LoadJSON("./fixture/scheduler_job.json")
	resp, err := BasicAuthPost(server.URL+"/v2/jobs", "application/json", bytes.NewBuffer(jsonStr))
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode, "Response should be success")
	respJSON := JobJSONFromResponse(resp.Body)

	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		app.Store.Where("JobID", respJSON.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))

	var initr models.Initiator
	app.Store.One("JobID", respJSON.ID, &initr)
	assert.Equal(t, "cron", initr.Type)
	assert.Equal(t, "* * * * *", string(initr.Schedule), "Wrong cron schedule saved")
}
func TestCreateJobIntegration(t *testing.T) {
	RegisterTestingT(t)

	config := NewConfig()
	AddPrivateKey(config, "./fixture/3cb8e3fd9d27e39a5e9e6852b0e96160061fd4ea.json")
	app := NewApplicationWithConfig(config)
	assert.Nil(t, app.Store.KeyStore.Unlock(Password))
	eth := app.MockEthClient()
	server := app.NewServer()
	app.Start()
	defer app.Stop()

	defer CloseGock(t)
	gock.EnableNetworking()

	tickerResponse := `{"high": "10744.00", "last": "10583.75", "timestamp": "1512156162", "bid": "10555.13", "vwap": "10097.98", "volume": "17861.33960013", "low": "9370.11", "ask": "10583.00", "open": "9927.29"}`
	gock.New("https://www.bitstamp.net").
		Get("/api/ticker/").
		Reply(200).
		JSON(tickerResponse)

	eth.Register("eth_getTransactionCount", `0x0100`)
	hash, err := utils.StringToHash("0x83c52c31cd40a023728fbc21a570316acd4f90525f81f1d7c477fd958ffa467f")
	assert.Nil(t, err)
	sentAt := uint64(23456)
	confirmed := sentAt + 1
	safe := confirmed + config.EthMinConfirmations
	eth.Register("eth_blockNumber", utils.Uint64ToHex(sentAt))
	eth.Register("eth_sendRawTransaction", hash)
	eth.Register("eth_blockNumber", utils.Uint64ToHex(confirmed))
	eth.Register("eth_getTransactionReceipt", store.TxReceipt{})
	eth.Register("eth_blockNumber", utils.Uint64ToHex(safe))
	eth.Register("eth_getTransactionReceipt", store.TxReceipt{Hash: hash, BlockNumber: confirmed})

	jsonStr := LoadJSON("./fixture/job_integration.json")
	resp, err := BasicAuthPost(server.URL+"/v2/jobs", "application/json", bytes.NewBuffer(jsonStr))
	assert.Nil(t, err)
	defer resp.Body.Close()
	jobID := JobJSONFromResponse(resp.Body).ID

	url := server.URL + "/v2/jobs/" + jobID + "/runs"
	resp, err = BasicAuthPost(url, "application/json", &bytes.Buffer{})
	assert.Nil(t, err)
	jrID := JobJSONFromResponse(resp.Body).ID

	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		app.Store.Where("JobID", jobID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))

	var job models.Job
	err = app.Store.One("ID", jobID, &job)
	assert.Nil(t, err)

	jobRuns, err = app.Store.JobRunsFor(job)
	assert.Nil(t, err)
	jobRun := jobRuns[0]
	assert.Equal(t, jrID, jobRun.ID)
	Eventually(func() string {
		assert.Nil(t, app.Store.One("ID", jobRun.ID, &jobRun))
		return jobRun.Status
	}).Should(Equal("completed"))
	assert.Equal(t, tickerResponse, jobRun.TaskRuns[0].Result.Value())
	assert.Equal(t, "10583.75", jobRun.TaskRuns[1].Result.Value())
	assert.Equal(t, hash.String(), jobRun.TaskRuns[3].Result.Value())
	assert.Equal(t, hash.String(), jobRun.Result.Value())
}

func TestCreateJobWithRunAtIntegration(t *testing.T) {
	RegisterTestingT(t)
	t.Parallel()
	app := NewApplication()
	app.InstantClock()
	server := app.NewServer()
	defer app.Stop()

	jsonStr := LoadJSON("./fixture/run_at_jobs.json")
	resp, _ := BasicAuthPost(server.URL+"v2/jobs", "application/json", bytes.NewBuffer(jsonStr))
	respJSON := JobJSONFromResponse(resp.Body)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Response should be success")
	var j models.Job
	app.Store.One("ID", respJSON.ID, &j)

	var initr models.Initiator
	app.Store.One("JobID", j.ID, &initr)
	assert.Equal(t, "runAt", initr.Type)
	assert.Equal(t, "2018-01-08T18:12:01Z", initr.Time.ISO8601())

	app.Start()
	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		app.Store.Where("JobID", respJSON.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))
}

func TestCreateInvalidTasks(t *testing.T) {
	t.Parallel()
	//fixtureprepare.SetUpDB()
	//defer fixtureprepare.TearDownDB()
	app := NewApplication()
	server := app.NewServer()
	defer app.Stop()

	jsonStr := LoadJSON("./fixture/invalid_job.json")
	resp, err := BasicAuthPost(server.URL+"/v2/jobs", "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 500, resp.StatusCode, "Response should be internal error")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, `{"errors":["IdoNotExist is not a supported adapter type"]}`, string(body), "Repsonse should return JSON")
}

func TestCreateInvalidCron(t *testing.T) {
	t.Parallel()
	app := NewApplication()
	server := app.NewServer()
	defer app.Stop()

	jsonStr := LoadJSON("./fixture/invalid_cron.json")
	resp, err := BasicAuthPost(server.URL+"/v2/jobs", "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 500, resp.StatusCode, "Response should be internal error")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	assert.Equal(t, `{"errors":["Cron: Failed to parse int from !: strconv.Atoi: parsing \"!\": invalid syntax"]}`, string(body), "Response should return JSON")
}

func TestShowJobs(t *testing.T) {
	t.Parallel()
	app := NewApplication()
	server := app.NewServer()
	defer app.Stop()

	j := NewJobWithSchedule("*****")
	app.Store.Save(&j)
	jr := j.NewRun()
	app.Store.Save(&jr)

	resp, err := BasicAuthGet(server.URL + "/v2/jobs/" + j.ID)
	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "Response should be successful")
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	var respJob controllers.JobPresenter
	json.Unmarshal(b, &respJob)
	assert.Equal(t, respJob.Initiators[0].Schedule, j.Initiators[0].Schedule, "should have the same schedule")
	assert.Equal(t, respJob.Runs[0].ID, jr.ID, "should have the job runs")
}

func TestShowNotFoundJobs(t *testing.T) {
	t.Parallel()
	app := NewApplication()
	server := app.NewServer()
	defer app.Stop()
	resp, err := BasicAuthGet(server.URL + "/v2/jobs/" + "garbage")
	assert.Nil(t, err)
	assert.Equal(t, 404, resp.StatusCode, "Response should be not found")
}

func TestShowJobUnauthenticated(t *testing.T) {
	t.Parallel()
	app := NewApplication()
	server := app.NewServer()
	defer app.Stop()

	resp, err := http.Get(server.URL + "/v2/jobs/" + "garbage")
	assert.Nil(t, err)
	assert.Equal(t, 401, resp.StatusCode, "Response should be forbidden")
}


func TestCreateJobWithEthLogIntegration(t *testing.T) {
	RegisterTestingT(t)
	t.Parallel()
	app := NewApplication()
	server := app.NewServer()
	eth := app.MockEthClient()
	defer app.Stop()

	jsonStr := LoadJSON("./fixture/eth_log_job.json")
	address, _ := utils.StringToAddress("0x3cCad4715152693fE3BC4460591e3D3Fbd071b42")
	resp, _ := BasicAuthPost(
		server.URL+"/v2/jobs",
		"application/json",
		bytes.NewBuffer(jsonStr),
	)
	respJSON := JobJSONFromResponse(resp.Body)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode, "Response should be success")
	var j models.Job
	app.Store.One("ID", respJSON.ID, &j)

	var initr models.Initiator
	app.Store.One("JobID", j.ID, &initr)
	assert.Equal(t, "ethLog", initr.Type)
	assert.Equal(t, address, initr.Address)

	logs := make(chan store.EventLog, 1)
	eth.RegisterSubscription("logs", logs)
	app.Start()

	logs <- store.EventLog{Address: address}

	jobRuns := []models.JobRun{}
	Eventually(func() []models.JobRun {
		app.Store.Where("JobID", respJSON.ID, &jobRuns)
		return jobRuns
	}).Should(HaveLen(1))
}

