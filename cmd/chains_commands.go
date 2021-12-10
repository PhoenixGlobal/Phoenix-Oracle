package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"PhoenixOracle/web/presenters"
	"github.com/urfave/cli"
	"go.uber.org/multierr"
)

type ChainPresenter struct {
	presenters.ChainResource
}

func (p *ChainPresenter) ToRow() []string {
	config, err := json.MarshalIndent(p.Config, "", "    ")
	if err != nil {
		panic(err)
	}

	row := []string{
		p.GetID(),
		string(config),
		p.CreatedAt.String(),
		p.UpdatedAt.String(),
	}
	return row
}

type ChainPresenters []ChainPresenter

// RenderTable implements TableRenderer
func (ps ChainPresenters) RenderTable(rt RendererTable) error {
	headers := []string{"ID", "Config", "Created", "Updated"}
	rows := [][]string{}

	for _, p := range ps {
		rows = append(rows, p.ToRow())
	}

	renderList(headers, rows, rt.Writer)

	return nil
}

func (cli *Client) IndexChains(c *cli.Context) (err error) {
	return cli.getPage("/v2/chains/evm", c.Int("page"), &ChainPresenters{})
}

func (cli *Client) CreateChain(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("must pass in the chain's parameters [-id integer] [JSON blob | JSON filepath]"))
	}
	chainID := c.Int64("id")
	if chainID == 0 {
		return cli.errorOut(errors.New("missing chain ID [-id integer]"))
	}

	buf, err := getBufferFromJSON(c.Args().First())
	if err != nil {
		return cli.errorOut(err)
	}

	params := map[string]interface{}{
		"chainID": chainID,
		"config":  buf,
	}

	body, err := json.Marshal(params)
	if err != nil {
		return cli.errorOut(err)
	}

	resp, err := cli.HTTP.Post("/v2/chains/evm", bytes.NewBuffer(body))
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	return cli.renderAPIResponse(resp, &ChainPresenter{})
}

func (cli *Client) RemoveChain(c *cli.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("must pass the id of the chain to be removed"))
	}
	chainID := c.Args().First()
	resp, err := cli.HTTP.Delete("/v2/chains/evm/" + chainID)
	if err != nil {
		return cli.errorOut(err)
	}
	_, err = cli.parseResponse(resp)
	if err != nil {
		return cli.errorOut(err)
	}

	fmt.Printf("Chain %v deleted\n", c.Args().First())
	return nil
}
