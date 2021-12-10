package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/manyminds/api2go/jsonapi"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	clipkg "github.com/urfave/cli"
	"go.uber.org/multierr"

	"PhoenixOracle/db/models"
	"PhoenixOracle/db/presenters"
	"PhoenixOracle/web"
	"PhoenixOracle/web/controllers"
	webpresenters "PhoenixOracle/web/presenters"
)

var errUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))

func (cli *Client) CreateExternalInitiator(c *clipkg.Context) (err error) {
	if c.NArg() != 1 && c.NArg() != 2 {
		return cli.errorOut(errors.New("create expects 1 - 2 arguments: a name and a url (optional)"))
	}

	var request models.ExternalInitiatorRequest
	request.Name = c.Args().Get(0)

	// process optional URL
	if c.NArg() == 2 {
		var reqURL *url.URL
		reqURL, err = url.ParseRequestURI(c.Args().Get(1))
		if err != nil {
			return cli.errorOut(err)
		}
		request.URL = (*models.WebURL)(reqURL)
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return cli.errorOut(err)
	}

	buf := bytes.NewBuffer(requestData)
	resp, err := cli.HTTP.Post("/v2/external_initiators", buf)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var ei webpresenters.ExternalInitiatorAuthentication
	err = cli.renderAPIResponse(resp, &ei)
	return err
}

func (cli *Client) DeleteExternalInitiator(c *clipkg.Context) (err error) {
	if !c.Args().Present() {
		return cli.errorOut(errors.New("Must pass the name of the external initiator to delete"))
	}

	resp, err := cli.HTTP.Delete("/v2/external_initiators/" + c.Args().First())
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()
	_, err = cli.parseResponse(resp)
	return err
}

func (cli *Client) getPage(requestURI string, page int, model interface{}) (err error) {
	uri, err := url.Parse(requestURI)
	if err != nil {
		return err
	}
	q := uri.Query()
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}
	uri.RawQuery = q.Encode()

	resp, err := cli.HTTP.Get(uri.String())
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	err = cli.deserializeAPIResponse(resp, model, &jsonapi.Links{})
	if err != nil {
		return err
	}
	err = cli.errorOut(cli.Render(model))
	return err
}

func (cli *Client) ReplayFromBlock(c *clipkg.Context) (err error) {

	blockNumber := c.Int64("block-number")
	if blockNumber <= 0 {
		return cli.errorOut(errors.New("Must pass a positive value in '--block-number' parameter"))
	}

	buf := bytes.NewBufferString("{}")

	resp, err := cli.HTTP.Post(fmt.Sprintf("/v2/replay_from_block/%v", blockNumber), buf)
	if err != nil {
		return cli.errorOut(err)
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bytes, err2 := cli.parseResponse(resp)
		if err2 != nil {
			return errors.Wrap(err2, "parseResponse error")
		}
		return cli.errorOut(errors.New(string(bytes)))
	}

	err = cli.printResponseBody(resp)
	return err
}

func (cli *Client) RemoteLogin(c *clipkg.Context) error {
	sessionRequest, err := cli.buildSessionRequest(c.String("file"))
	if err != nil {
		return cli.errorOut(err)
	}
	_, err = cli.CookieAuthenticator.Authenticate(sessionRequest)
	return cli.errorOut(err)
}

func (cli *Client) ChangePassword(c *clipkg.Context) (err error) {
	req, err := cli.ChangePasswordPrompter.Prompt()
	if err != nil {
		return cli.errorOut(err)
	}

	requestData, err := json.Marshal(req)
	if err != nil {
		return cli.errorOut(err)
	}

	buf := bytes.NewBuffer(requestData)
	resp, err := cli.HTTP.Patch("/v2/user/password", buf)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Println("Password updated.")
	case http.StatusConflict:
		fmt.Println("Old password did not match.")
	default:
		return cli.printResponseBody(resp)
	}
	return nil
}

func (cli *Client) buildSessionRequest(flag string) (models.SessionRequest, error) {
	if len(flag) > 0 {
		return cli.FileSessionRequestBuilder.Build(flag)
	}
	return cli.PromptingSessionRequestBuilder.Build("")
}

func getTOMLString(s string) (string, error) {
	var val interface{}
	err := toml.Unmarshal([]byte(s), &val)
	if err == nil {
		return s, nil
	}

	buf, err := fromFile(s)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("invalid TOML or file not found '%s'", s)
	} else if err != nil {
		return "", fmt.Errorf("error reading from file '%s': %v", s, err)
	}
	return buf.String(), nil
}

func (cli *Client) parseResponse(resp *http.Response) ([]byte, error) {
	b, err := parseResponse(resp)
	if err == errUnauthorized {
		return nil, cli.errorOut(multierr.Append(err, fmt.Errorf("you must first login through the CLI")))
	}
	if err != nil {
		jae := models.JSONAPIErrors{}
		unmarshalErr := json.Unmarshal(b, &jae)
		return nil, cli.errorOut(multierr.Combine(err, unmarshalErr, &jae))
	}
	return b, err
}

func (cli *Client) printResponseBody(resp *http.Response) error {
	b, err := parseResponse(resp)
	if err != nil {
		return cli.errorOut(err)
	}

	fmt.Println(string(b))
	return nil
}

func (cli *Client) renderAPIResponse(resp *http.Response, dst interface{}, headers ...string) error {
	var links jsonapi.Links
	if err := cli.deserializeAPIResponse(resp, dst, &links); err != nil {
		return cli.errorOut(err)
	}

	return cli.errorOut(cli.Render(dst, headers...))
}

func (cli *Client) SetMinimumGasPrice(c *clipkg.Context) (err error) {
	if c.NArg() != 1 {
		return cli.errorOut(errors.New("expecting an amount"))
	}

	value := c.Args().Get(0)
	amount, ok := new(big.Float).SetString(value)
	if !ok {
		return cli.errorOut(fmt.Errorf("invalid ethereum amount %s", value))
	}

	if c.IsSet("gwei") {
		amount.Mul(amount, big.NewFloat(1000000000))
	}

	adjustedAmount, _ := amount.Int(nil)
	request := struct {
		EvmGasPriceDefault string `json:"ethGasPriceDefault"`
	}{EvmGasPriceDefault: adjustedAmount.String()}
	requestData, err := json.Marshal(request)
	if err != nil {
		return cli.errorOut(err)
	}

	buf := bytes.NewBuffer(requestData)
	response, err := cli.HTTP.Patch("/v2/config", buf)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := response.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	patchResponse := controllers.ConfigPatchResponse{}
	if err = cli.deserializeAPIResponse(response, &patchResponse, &jsonapi.Links{}); err != nil {
		return err
	}

	err = cli.errorOut(cli.Render(&patchResponse))
	return err
}

func (cli *Client) GetConfiguration(c *clipkg.Context) (err error) {
	resp, err := cli.HTTP.Get("/v2/config")
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()
	cwl := presenters.ConfigPrinter{}
	err = cli.renderAPIResponse(resp, &cwl)
	return err
}

func normalizePassword(password string) string {
	return url.PathEscape(strings.TrimSpace(password))
}

func (cli *Client) SetLogLevel(c *clipkg.Context) (err error) {
	logLevel := c.String("level")
	request := controllers.LogPatchRequest{Level: logLevel}
	requestData, err := json.Marshal(request)
	if err != nil {
		return cli.errorOut(err)
	}

	buf := bytes.NewBuffer(requestData)
	resp, err := cli.HTTP.Patch("/v2/log", buf)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var svcLogConfig webpresenters.ServiceLogConfigResource
	err = cli.renderAPIResponse(resp, &svcLogConfig)
	return err
}

func (cli *Client) SetLogSQL(c *clipkg.Context) (err error) {

	// Enforces selection of --enable or --disable
	if !c.Bool("enable") && !c.Bool("disable") {
		return cli.errorOut(errors.New("Must set logSql --enabled || --disable"))
	}

	// Sets logSql to true || false based on the --enabled flag
	logSql := c.Bool("enable")

	if err != nil {
		return cli.errorOut(err)
	}
	request := controllers.LogPatchRequest{SqlEnabled: &logSql}
	requestData, err := json.Marshal(request)
	if err != nil {
		return cli.errorOut(err)
	}

	buf := bytes.NewBuffer(requestData)
	resp, err := cli.HTTP.Patch("/v2/log", buf)
	if err != nil {
		return cli.errorOut(err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var svcLogConfig webpresenters.ServiceLogConfigResource
	err = cli.renderAPIResponse(resp, &svcLogConfig)
	return err
}

func (cli *Client) SetLogPkg(c *clipkg.Context) (err error) {
	pkg := strings.Split(c.String("pkg"), ",")
	level := strings.Split(c.String("level"), ",")

	serviceLogLevel := make([][2]string, len(pkg))
	for i, p := range pkg {
		serviceLogLevel[i][0] = p
		serviceLogLevel[i][1] = level[i]
	}

	request := controllers.LogPatchRequest{ServiceLogLevel: serviceLogLevel}
	requestData, err := json.Marshal(request)
	if err != nil {
		return cli.errorOut(err)
	}

	buf := bytes.NewBuffer(requestData)
	resp, err := cli.HTTP.Patch("/v2/log", buf)
	if err != nil {
		return cli.errorOut(errors.Wrap(err, "set pkg specific logging levels"))
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			err = multierr.Append(err, cerr)
		}
	}()

	var svcLogConfig webpresenters.ServiceLogConfigResource
	err = cli.renderAPIResponse(resp, &svcLogConfig)

	return err
}

func getBufferFromJSON(s string) (*bytes.Buffer, error) {
	if gjson.Valid(s) {
		return bytes.NewBufferString(s), nil
	}

	buf, err := fromFile(s)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("invalid JSON or file not found '%s'", s)
	} else if err != nil {
		return nil, fmt.Errorf("error reading from file '%s': %v", s, err)
	}
	return buf, nil
}

func fromFile(arg string) (*bytes.Buffer, error) {
	dir, err := homedir.Expand(arg)
	if err != nil {
		return nil, err
	}
	file, err := ioutil.ReadFile(dir)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(file), nil
}

func (cli *Client) deserializeAPIResponse(resp *http.Response, dst interface{}, links *jsonapi.Links) error {
	b, err := cli.parseResponse(resp)
	if err != nil {
		return errors.Wrap(err, "parseResponse error")
	}
	if err = web.ParsePaginatedResponse(b, dst, links); err != nil {
		return cli.errorOut(err)
	}
	return nil
}

func parseResponse(resp *http.Response) ([]byte, error) {
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return b, multierr.Append(errors.New(resp.Status), err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return b, errUnauthorized
	} else if resp.StatusCode >= http.StatusBadRequest {
		return b, errors.New("Error")
	}
	return b, err
}
