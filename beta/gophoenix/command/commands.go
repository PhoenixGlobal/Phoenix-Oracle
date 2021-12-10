package command

import (
	"PhoenixOracle/gophoenix/core/logger"
	"PhoenixOracle/gophoenix/core/services"
	"PhoenixOracle/gophoenix/core/store"
	"PhoenixOracle/gophoenix/core/store/models"
	"PhoenixOracle/gophoenix/core/utils"
	"PhoenixOracle/gophoenix/core/web"
	"PhoenixOracle/gophoenix/core/web/controllers"
	"encoding/json"
	"errors"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	io.Writer
}

func (self *Client) PrettyPrintJSON(v interface{}) error {
	b, err := utils.FormatJSON(v)
	if err != nil {
		return err
	}
	if _, err = self.Write(b); err != nil {
		return err
	}
	return nil
}

func (self *Client) RunNode(c *cli.Context) error {
	cl := services.NewApplication(store.NewConfig())
	services.Authenticate(cl.Store)
	r := web.Router(cl)

	if err := cl.Start(); err != nil {
		logger.Fatal(err)
	}
	defer cl.Stop()
	logger.Fatal(r.Run())
	return nil
}

func (self *Client) ShowJob(c *cli.Context) error {
	cfg := store.NewConfig()
	if !c.Args().Present() {
		return self.cliError(errors.New("Must pass the job id to be shown"))
	}
	resp, err := utils.BasicAuthGet(
		cfg.BasicAuthUsername,
		cfg.BasicAuthPassword,
		"http://localhost:8080/jobs/"+c.Args().First(),
	)
	if err != nil {
		return self.cliError(err)
	}
	defer resp.Body.Close()
	var job controllers.JobPresenter
	return self.deserializeResponse(resp, &job)
}

func (self *Client) GetJobs(c *cli.Context) error {
	cfg := store.NewConfig()
	resp, err := utils.BasicAuthGet(
		cfg.BasicAuthUsername,
		cfg.BasicAuthPassword,
		"http://localhost:8080/jobs",
	)
	if err != nil {
		return self.cliError(err)
	}
	defer resp.Body.Close()

	var jobs []models.Job
	return self.deserializeResponse(resp, &jobs)
}

func (self *Client) deserializeResponse(resp *http.Response, dst interface{}) error {
	if resp.StatusCode >= 300 {
		return self.cliError(errors.New(resp.Status))
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return self.cliError(err)
	}
	if err = json.Unmarshal(b, &dst); err != nil {
		return self.cliError(err)
	}
	return self.cliError(self.PrettyPrintJSON(dst))
}

func (self *Client) cliError(err error) error {
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}
