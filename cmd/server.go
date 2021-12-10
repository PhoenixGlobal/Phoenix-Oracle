package cmd

import (
	"PhoenixOracle/db"
	"PhoenixOracle/db/models"
	"PhoenixOracle/lib/logger"
	"PhoenixOracle/web"
	"PhoenixOracle/web/controllers"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	clipkg "github.com/urfave/cli"
	"go.uber.org/multierr"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func runServer(handler *gin.Engine, port uint16, writeTimeout time.Duration) error {
	logger.Infof("Listening and serving HTTP on port %d", port)
	server := createServer(handler, port, writeTimeout)
	err := server.ListenAndServe()
	logger.ErrorIf(err)
	return err
}

func runServerTLS(handler *gin.Engine, port uint16, certFile, keyFile string, writeTimeout time.Duration) error {
	logger.Infof("Listening and serving HTTPS on port %d", port)
	server := createServer(handler, port, writeTimeout)
	err := server.ListenAndServeTLS(certFile, keyFile)
	logger.ErrorIf(err)
	return err
}

func createServer(handler *gin.Engine, port uint16, writeTimeout time.Duration) *http.Server {
	url := fmt.Sprintf(":%d", port)
	s := &http.Server{
		Addr:           url,
		Handler:        handler,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return s
}

// HTTPClient encapsulates all methods used to interact with a phoenix node API.
type HTTPClient interface {
	Get(string, ...map[string]string) (*http.Response, error)
	Post(string, io.Reader) (*http.Response, error)
	Put(string, io.Reader) (*http.Response, error)
	Patch(string, io.Reader, ...map[string]string) (*http.Response, error)
	Delete(string) (*http.Response, error)
}

type HTTPClientConfig interface {
	SessionCookieAuthenticatorConfig
}

type authenticatedHTTPClient struct {
	config         HTTPClientConfig
	client         *http.Client
	cookieAuth     CookieAuthenticator
	sessionRequest models.SessionRequest
}

func NewAuthenticatedHTTPClient(config HTTPClientConfig, cookieAuth CookieAuthenticator, sessionRequest models.SessionRequest) HTTPClient {
	return &authenticatedHTTPClient{
		config:         config,
		client:         newHttpClient(config),
		cookieAuth:     cookieAuth,
		sessionRequest: sessionRequest,
	}
}

func newHttpClient(config SessionCookieAuthenticatorConfig) *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.InsecureSkipVerify()},
	}
	if config.InsecureSkipVerify() {
		fmt.Println("WARNING: INSECURE_SKIP_VERIFY is set to true, skipping SSL certificate verification.")
	}
	return &http.Client{Transport: tr}
}

func (h *authenticatedHTTPClient) Get(path string, headers ...map[string]string) (*http.Response, error) {
	return h.doRequest("GET", path, nil, headers...)
}

func (h *authenticatedHTTPClient) Post(path string, body io.Reader) (*http.Response, error) {
	return h.doRequest("POST", path, body)
}

func (h *authenticatedHTTPClient) Put(path string, body io.Reader) (*http.Response, error) {
	return h.doRequest("PUT", path, body)
}

func (h *authenticatedHTTPClient) Patch(path string, body io.Reader, headers ...map[string]string) (*http.Response, error) {
	return h.doRequest("PATCH", path, body, headers...)
}

func (h *authenticatedHTTPClient) Delete(path string) (*http.Response, error) {
	return h.doRequest("DELETE", path, nil)
}

func (h *authenticatedHTTPClient) doRequest(verb, path string, body io.Reader, headerArgs ...map[string]string) (*http.Response, error) {
	var headers map[string]string
	if len(headerArgs) > 0 {
		headers = headerArgs[0]
	} else {
		headers = map[string]string{}
	}

	request, err := http.NewRequest(verb, h.config.ClientNodeURL()+path, body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		request.Header.Add(key, value)
	}
	cookie, err := h.cookieAuth.Cookie()
	if err != nil {
		return nil, err
	} else if cookie != nil {
		request.AddCookie(cookie)
	}

	response, err := h.client.Do(request)
	if err != nil {
		return response, err
	}
	if response.StatusCode == http.StatusUnauthorized && (h.sessionRequest.Email != "" || h.sessionRequest.Password != "") {
		var cookieerr error
		cookie, cookieerr = h.cookieAuth.Authenticate(h.sessionRequest)
		if cookieerr != nil {
			return response, err
		}
		request.Header.Set("Cookie", "")
		request.AddCookie(cookie)
		response, err = h.client.Do(request)
		if err != nil {
			return response, err
		}
	}
	return response, nil
}

type CookieAuthenticator interface {
	Cookie() (*http.Cookie, error)
	Authenticate(models.SessionRequest) (*http.Cookie, error)
}

type SessionCookieAuthenticatorConfig interface {
	ClientNodeURL() string
	InsecureSkipVerify() bool
}

type SessionCookieAuthenticator struct {
	config SessionCookieAuthenticatorConfig
	store  CookieStore
}

func NewSessionCookieAuthenticator(config SessionCookieAuthenticatorConfig, store CookieStore) CookieAuthenticator {
	return &SessionCookieAuthenticator{config: config, store: store}
}

func (t *SessionCookieAuthenticator) Cookie() (*http.Cookie, error) {
	return t.store.Retrieve()
}

func (t *SessionCookieAuthenticator) Authenticate(sessionRequest models.SessionRequest) (*http.Cookie, error) {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(sessionRequest)
	if err != nil {
		return nil, err
	}
	url := t.config.ClientNodeURL() + "/sessions"
	req, err := http.NewRequest("POST", url, b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := newHttpClient(t.config)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer logger.ErrorIfCalling(resp.Body.Close)

	_, err = parseResponse(resp)
	if err != nil {
		return nil, err
	}

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		return nil, errors.New("did not receive cookie with session id")
	}
	sc := web.FindSessionCookie(cookies)
	return sc, t.store.Save(sc)
}

type CookieStore interface {
	Save(cookie *http.Cookie) error
	Retrieve() (*http.Cookie, error)
}

type MemoryCookieStore struct {
	Cookie *http.Cookie
}

// Save stores a cookie.
func (m *MemoryCookieStore) Save(cookie *http.Cookie) error {
	m.Cookie = cookie
	return nil
}

func (m *MemoryCookieStore) Retrieve() (*http.Cookie, error) {
	return m.Cookie, nil
}

type DiskCookieConfig interface {
	RootDir() string
}

type DiskCookieStore struct {
	Config DiskCookieConfig
}

func (d DiskCookieStore) Save(cookie *http.Cookie) error {
	return ioutil.WriteFile(d.cookiePath(), []byte(cookie.String()), 0600)
}

func (d DiskCookieStore) Retrieve() (*http.Cookie, error) {
	b, err := ioutil.ReadFile(d.cookiePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, multierr.Append(errors.New("unable to retrieve credentials, you must first login through the CLI"), err)
	}
	header := http.Header{}
	header.Add("Cookie", string(b))
	request := http.Request{Header: header}
	cookies := request.Cookies()
	if len(cookies) == 0 {
		return nil, errors.New("Cookie not in file, you must first login through the CLI")
	}
	return request.Cookies()[0], nil
}

func (d DiskCookieStore) cookiePath() string {
	return path.Join(d.Config.RootDir(), "cookie")
}

type SessionRequestBuilder interface {
	Build(flag string) (models.SessionRequest, error)
}

type promptingSessionRequestBuilder struct {
	prompter Prompter
}

func NewPromptingSessionRequestBuilder(prompter Prompter) SessionRequestBuilder {
	return promptingSessionRequestBuilder{prompter}
}

func (p promptingSessionRequestBuilder) Build(string) (models.SessionRequest, error) {
	email := p.prompter.Prompt("Enter email: ")
	pwd := p.prompter.PasswordPrompt("Enter password: ")
	return models.SessionRequest{Email: email, Password: pwd}, nil
}

type fileSessionRequestBuilder struct{}

func NewFileSessionRequestBuilder() SessionRequestBuilder {
	return fileSessionRequestBuilder{}
}

func (f fileSessionRequestBuilder) Build(file string) (models.SessionRequest, error) {
	return credentialsFromFile(file)
}

type APIInitializer interface {
	// Initialize creates a new user for API access, or does nothing if one exists.
	Initialize(store *db.Store) (models.User, error)
}

type promptingAPIInitializer struct {
	prompter Prompter
}

func NewPromptingAPIInitializer(prompter Prompter) APIInitializer {
	return &promptingAPIInitializer{prompter: prompter}
}

func (t *promptingAPIInitializer) Initialize(store *db.Store) (models.User, error) {
	if user, err := store.FindUser(); err == nil {
		return user, err
	}

	if !t.prompter.IsTerminal() {
		return models.User{}, ErrorNoAPICredentialsAvailable
	}

	for {
		email := t.prompter.Prompt("Enter API Email: ")
		pwd := t.prompter.PasswordPrompt("Enter API Password: ")
		user, err := models.NewUser(email, pwd)
		if err != nil {
			fmt.Println("Error creating API user: ", err)
			continue
		}
		if err = store.SaveUser(&user); err != nil {
			fmt.Println("Error creating API user: ", err)
		}
		return user, err
	}
}

type fileAPIInitializer struct {
	file string
}

func NewFileAPIInitializer(file string) APIInitializer {
	return fileAPIInitializer{file: file}
}

func (f fileAPIInitializer) Initialize(store *db.Store) (models.User, error) {
	if user, err := store.FindUser(); err == nil {
		return user, err
	}

	request, err := credentialsFromFile(f.file)
	if err != nil {
		return models.User{}, err
	}

	user, err := models.NewUser(request.Email, request.Password)
	if err != nil {
		return user, err
	}
	return user, store.SaveUser(&user)
}

var ErrNoCredentialFile = errors.New("no API user credential file was passed")

func credentialsFromFile(file string) (models.SessionRequest, error) {
	if len(file) == 0 {
		return models.SessionRequest{}, ErrNoCredentialFile
	}

	logger.Debug("Initializing API credentials from ", file)
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return models.SessionRequest{}, err
	}
	lines := strings.Split(string(dat), "\n")
	if len(lines) < 2 {
		return models.SessionRequest{}, fmt.Errorf("malformed API credentials file does not have at least two lines at %s", file)
	}
	credentials := models.SessionRequest{
		Email:    strings.TrimSpace(lines[0]),
		Password: strings.TrimSpace(lines[1]),
	}
	return credentials, nil
}

type ChangePasswordPrompter interface {
	Prompt() (controllers.UpdatePasswordRequest, error)
}

func NewChangePasswordPrompter() ChangePasswordPrompter {
	prompter := NewTerminalPrompter()
	return changePasswordPrompter{prompter: prompter}
}

type changePasswordPrompter struct {
	prompter Prompter
}

func (c changePasswordPrompter) Prompt() (controllers.UpdatePasswordRequest, error) {
	fmt.Println("Changing your phoenix account password.")
	fmt.Println("NOTE: This will terminate any other sessions.")
	oldPassword := c.prompter.PasswordPrompt("Password:")

	fmt.Println("Now enter your **NEW** password")
	newPassword := c.prompter.PasswordPrompt("Password:")
	confirmPassword := c.prompter.PasswordPrompt("Confirmation:")

	if newPassword != confirmPassword {
		return controllers.UpdatePasswordRequest{}, errors.New("new password and confirmation did not match")
	}

	return controllers.UpdatePasswordRequest{
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}, nil
}

type PasswordPrompter interface {
	Prompt() string
}

func NewPasswordPrompter() PasswordPrompter {
	prompter := NewTerminalPrompter()
	return passwordPrompter{prompter: prompter}
}

type passwordPrompter struct {
	prompter Prompter
}

func (c passwordPrompter) Prompt() string {
	return c.prompter.PasswordPrompt("Password:")
}

func confirmAction(c *clipkg.Context) bool {
	if len(c.String("yes")) > 0 {
		yes, err := strconv.ParseBool(c.String("yes"))
		if err == nil && yes {
			return true
		}
	}

	prompt := NewTerminalPrompter()
	var answer string
	for {
		answer = prompt.Prompt("Are you sure? This action is irreversible! (yes/no) ")
		if answer == "yes" {
			return true
		} else if answer == "no" {
			return false
		} else {
			fmt.Printf("%s is not valid. Please type yes or no\n", answer)
		}
	}
}