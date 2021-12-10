package cmd

import (
	"PhoenixOracle/core/service/ethereum"
	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/config"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/lib/postgres"
	"PhoenixOracle/web/controllers"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	clipkg "github.com/urfave/cli"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/errgroup"
	"log"
)

var (
	ErrorNoAPICredentialsAvailable = errors.New("API credentials must be supplied")
)

type Client struct {
	Renderer
	Config                         config.GeneralConfig
	AppFactory                     AppFactory
	KeyStoreAuthenticator          TerminalKeyStoreAuthenticator
	FallbackAPIInitializer         APIInitializer
	Runner                         Runner
	HTTP                           HTTPClient
	CookieAuthenticator            CookieAuthenticator
	FileSessionRequestBuilder      SessionRequestBuilder
	PromptingSessionRequestBuilder SessionRequestBuilder
	ChangePasswordPrompter         ChangePasswordPrompter
	PasswordPrompter               PasswordPrompter
}

func (cli *Client) errorOut(err error) error {
	if err != nil {
		return clipkg.NewExitError(err.Error(), 1)
	}
	return nil
}

type AppFactory interface {
	NewApplication(config.EVMConfig) (phoenix.Application, error)
}

// used to create a new Application.
type PhoenixAppFactory struct{}

func (n PhoenixAppFactory) NewApplication(config config.EVMConfig) (phoenix.Application, error) {
	chainLogger := logger.Default.With(
		"chainId", config.Chain().ID(),
	)

	var ethClient ethereum.Client
	if config.EthereumDisabled() {
		ethClient = &ethereum.NullClient{}
	} else {
		var err error
		ethClient, err = ethereum.NewClient(chainLogger, config.EthereumURL(), config.EthereumHTTPURL(), config.EthereumSecondaryURLs())
		if err != nil {
			return nil, err
		}
	}

	advisoryLock := postgres.NewAdvisoryLock(config.DatabaseURL())
	return phoenix.NewApplication(chainLogger, config, ethClient, advisoryLock)
}

// implements the Run method.
type Runner interface {
	Run(phoenix.Application) error
}

// used to run the node application.
type PhoenixRunner struct{}

func (n PhoenixRunner) Run(app phoenix.Application) error {
	config := app.GetStore().Config
	mode := gin.ReleaseMode
	if config.Dev() && config.LogLevel().Level < zapcore.InfoLevel {
		mode = gin.DebugMode
	}
	gin.SetMode(mode)
	handler := controllers.Router(app.(*phoenix.PhoenixApplication))
	var g errgroup.Group

	if config.Port() == 0 && config.TLSPort() == 0 {
		log.Fatal("You must specify at least one port to listen on")
	}

	if config.Port() != 0 {
		g.Go(func() error { return runServer(handler, config.Port(), config.HTTPServerWriteTimeout()) })
	}

	if config.TLSPort() != 0 {
		g.Go(func() error {
			return runServerTLS(
				handler,
				config.TLSPort(),
				config.CertFile(),
				config.KeyFile(),
				config.HTTPServerWriteTimeout())
		})
	}

	return g.Wait()
}