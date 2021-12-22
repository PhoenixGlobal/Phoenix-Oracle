package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"

	"PhoenixOracle/web/controllers"

	"PhoenixOracle/core/service/pipeline"
	"PhoenixOracle/web/presenters"
	"github.com/urfave/cli"
	"go.uber.org/multierr"
)

type JobPresenter struct {
	JAID // needed to render the id for a JSONAPI Resource as normal JSON
	presenters.JobResource
}

func (p JobPresenter) ToRows() [][]string {
	row := [][]string{}

	// Produce a row when there are no tasks
	if len(p.FriendlyTasks()) == 0 {
		row = append(row, p.toRow(""))

		return row
	}

	for _, t := range p.FriendlyTasks() {
		row = append(row, p.toRow(t))
	}

	return row
}

func (p JobPresenter) toRow(task string) []string {
	return []string{
		p.GetID(),
		p.Name,
		p.Type.String(),
		task,
		p.FriendlyCreatedAt(),
	}
}

func (p JobPresenter) GetTasks() ([]string, error) {
	types := []string{}
	pipeline, err := pipeline.Parse(p.PipelineSpec.DotDAGSource)
	if err != nil {
		return nil, err
	}

	for _, t := range pipeline.Tasks {
		types = append(types, fmt.Sprintf("%s %s", t.DotID(), t.Type()))
	}

	return types, nil
}

func (p JobPresenter) FriendlyTasks() []string {
	taskTypes, err := p.GetTasks()
	if err != nil {
		return []string{"error parsing DAG"}
	}

	return taskTypes
}

func (p JobPresenter) FriendlyCreatedAt() string {
	switch p.Type {
	case presenters.DirectRequestJobSpec:
		if p.DirectRequestSpec != nil {
			return p.DirectRequestSpec.CreatedAt.Format(time.RFC3339)
		}
	case presenters.FluxMonitorJobSpec:
		if p.FluxMonitorSpec != nil {
			return p.FluxMonitorSpec.CreatedAt.Format(time.RFC3339)
		}
	case presenters.OffChainReportingJobSpec:
		if p.OffChainReportingSpec != nil {
			return p.OffChainReportingSpec.CreatedAt.Format(time.RFC3339)
		}
	case presenters.KeeperJobSpec:
		if p.KeeperSpec != nil {
			return p.KeeperSpec.CreatedAt.Format(time.RFC3339)
		}
	case presenters.CronJobSpec:
		if p.CronSpec != nil {
			return p.CronSpec.CreatedAt.Format(time.RFC3339)
		}
	case presenters.VRFJobSpec:
		if p.VRFSpec != nil {
			return p.VRFSpec.CreatedAt.Format(time.RFC3339)
		}
	case presenters.WebhookJobSpec:
		if p.WebhookSpec != nil {
			return p.WebhookSpec.CreatedAt.Format(time.RFC3339)
		}
	default:
		return "unknown"
	}

	// This should never occur since the job should always have a spec matching
	// the type
	return "N/A"
}

func (p *JobPresenter) RenderTable(rt RendererTable) error {
	table := rt.newTable([]string{"ID", "Name", "Type", "Tasks", "Created At"})
	table.SetAutoMergeCells(true)
	for _, r := range p.ToRows() {
		table.Append(r)
	}

	render("Jobs (V2)", table)
	return nil
}

type JobPresenters []JobPresenter

func (ps JobPresenters) RenderTable(rt RendererTable) error {
	table := rt.newTable([]string{"ID", "Name", "Type", "Tasks", "Created At"})
	table.SetAutoMergeCells(true)
	for _, p := range ps {
		for _, r := range p.ToRows() {
			table.Append(r)
		}
	}

	render("Jobs (V2)", table)
	return nil
}

func (cli *Client) ListJobsV2(c *cli.Context) (err error) {
	return cli.getPage("/v2/jobs", c.Int("page"), &JobPresenters{})
}

func (cli *Client) CreateJobV2(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("must pass in TOML or filepath"))
	}

	tomlString, err := getTOMLString(c.Args().First())
	if err != nil {
		return cli.errorOut(err)
	}

	request, err := json.Marshal(controllers.CreateJobRequest{
		TOML: tomlString,
	})
	if err != nil {
		return cli.errorOut(err)
	}

	resp, err := cli.HTTP.Post("/v2/jobs", bytes.NewReader(request))
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	if resp.StatusCode >= 400 {
		body, rerr := ioutil.ReadAll(resp.Body)
		if err != nil {
			err = multierr.Append(err, rerr)
			return cli.errorOut(err)
		}
		fmt.Printf("Response: '%v', Status: %d\n", string(body), resp.StatusCode)
		return cli.errorOut(err)
	}

	err = cli.renderAPIResponse(resp, &JobPresenter{}, "Job created")
	return err
}

func (cli *Client) DeleteJob(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("must pass the job id to be archived"))
	}
	resp, err := cli.HTTP.Delete("/v2/jobs/" + c.Args().First())
	if err != nil {
		return cli.errorOut(err)
	}
	_, err = cli.parseResponse(resp)
	if err != nil {
		return cli.errorOut(err)
	}

	fmt.Printf("Job %v Deleted\n", c.Args().First())
	return nil
}

func (cli *Client) TriggerPipelineRun(c *cli.Context) error {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the job id to trigger a run"))
	}
	resp, err := cli.HTTP.Post("/v2/jobs/"+c.Args().First()+"/runs", nil)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var run presenters.PipelineRunResource
	err = cli.renderAPIResponse(resp, &run, "Pipeline run successfully triggered")
	return err
}
