package main

import (
	"os"

	"PhoenixOracle/cmd"
	"PhoenixOracle/db/config"
	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
	"github.com/pkg/errors"
)

func main() {
	Run(NewProductionClient(), os.Args...)
}

func Run(client *cmd.Client, args ...string) {
	app := cmd.NewApp(client)
	logger.WarnIf(app.Run(args))
}

func NewProductionClient() *cmd.Client {
	cfg := config.NewGeneralConfig()

	prompter := cmd.NewTerminalPrompter()
	cookieAuth := cmd.NewSessionCookieAuthenticator(cfg, cmd.DiskCookieStore{Config: cfg})
	sr := models.SessionRequest{}
	sessionRequestBuilder := cmd.NewFileSessionRequestBuilder()
	if credentialsFile := cfg.AdminCredentialsFile(); credentialsFile != "" {
		var err error
		sr, err = sessionRequestBuilder.Build(credentialsFile)
		if err != nil && errors.Cause(err) != cmd.ErrNoCredentialFile && !os.IsNotExist(err) {
			logger.Fatalw("Error loading API credentials", "error", err, "credentialsFile", credentialsFile)
		}
	}
	return &cmd.Client{
		Renderer:                       cmd.RendererTable{Writer: os.Stdout},
		Config:                         cfg,
		AppFactory:                     cmd.PhoenixAppFactory{},
		KeyStoreAuthenticator:          cmd.TerminalKeyStoreAuthenticator{Prompter: prompter},
		FallbackAPIInitializer:         cmd.NewPromptingAPIInitializer(prompter),
		Runner:                         cmd.PhoenixRunner{},
		HTTP:                           cmd.NewAuthenticatedHTTPClient(cfg, cookieAuth, sr),
		CookieAuthenticator:            cookieAuth,
		FileSessionRequestBuilder:      sessionRequestBuilder,
		PromptingSessionRequestBuilder: cmd.NewPromptingSessionRequestBuilder(prompter),
		ChangePasswordPrompter:         cmd.NewChangePasswordPrompter(),
		PasswordPrompter:               cmd.NewPasswordPrompter(),
	}
}
