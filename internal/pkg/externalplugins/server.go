package externalplugins

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
)

// Options contains common server options.
type ServerOptions struct {
	Port                  int
	DryRun                bool
	Github                prowflagutil.GitHubOptions
	ExternalPluginsConfig string
	WebhookSecretFile     string
}

func (o *ServerOptions) Validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.Github} {
		if err := group.Validate(o.DryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}

	return nil
}

func (o *ServerOptions) ParseFromFlags() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.Port, "port", 80, "Port to listen on.")
	fs.StringVar(&o.ExternalPluginsConfig, "external-plugins-config",
		"/etc/external_plugins_config/external_plugins_config.yaml", "Path to external plugin config file.")
	fs.BoolVar(&o.DryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.WebhookSecretFile, "hmac-secret-file",
		"/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")

	for _, group := range []flagutil.OptionGroup{&o.Github} {
		group.AddFlags(fs)
	}
	_ = fs.Parse(os.Args[1:])
}

// ParseOptions parses options from cli flags.
func ParseOptions(o *ServerOptions) *ServerOptions {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.Port, "port", 80, "Port to listen on.")
	fs.StringVar(&o.ExternalPluginsConfig, "external-plugins-config",
		"/etc/external_plugins_config/external_plugins_config.yaml", "Path to external plugin config file.")
	fs.BoolVar(&o.DryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.WebhookSecretFile, "hmac-secret-file",
		"/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")

	for _, group := range []flagutil.OptionGroup{&o.Github} {
		group.AddFlags(fs)
	}
	_ = fs.Parse(os.Args[1:])
	return o
}

type helpProviderGenerator func(cfg *ConfigAgent) externalplugins.ExternalPluginHelpProvider

// BaseServer implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type BaseServer struct {
	Options     *ServerOptions
	Log         *logrus.Entry
	ConfigAgent *ConfigAgent

	tokenGenerator func() []byte
	helpProvider   externalplugins.ExternalPluginHelpProvider
	eventHandlers  *EventHandlers
}

// GetExternalPluginConfig returns tide external config from configagent.
func (s *BaseServer) GetExternalPluginConfig() *Configuration {
	return s.ConfigAgent.Config()
}

// ServeHTTP implements http.Handler.
func (s *BaseServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}

	if err := DemuxEvent(s.Log, s.eventHandlers, eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

// SetHandlers sets event handlers to server.
func (s *BaseServer) SetHandlers(eventHandlers *EventHandlers) {
	s.eventHandlers = eventHandlers
}

// func (s *Provider) AddHandler(eventType EventType, )

// Run the server.
func (s *BaseServer) Run() {
	health := pjutil.NewHealth()
	health.ServeReady()

	mux := http.NewServeMux()
	mux.Handle("/", s)

	externalplugins.ServeExternalPluginHelp(mux, s.Log, s.helpProvider)

	httpServer := &http.Server{Addr: ":" + strconv.Itoa(s.Options.Port), Handler: mux}

	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)
}

// NewBaseServer returns a new Server.
func NewBaseServer(pluginName string, o *ServerOptions, helpProviderGenerator helpProviderGenerator) BaseServer {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
	log := logrus.StandardLogger().WithField("plugin", pluginName)

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.Github.TokenPath, o.WebhookSecretFile}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	epa := &ConfigAgent{}
	if err := epa.Start(o.ExternalPluginsConfig, false); err != nil {
		log.WithError(err).Fatalf("Error loading external plugin config from %q.", o.ExternalPluginsConfig)
	}

	githubClient, err := o.Github.GitHubClient(secretAgent, o.DryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	githubClient.Throttle(360, 360)
	server := BaseServer{
		Options:        o,
		ConfigAgent:    epa,
		Log:            log,
		tokenGenerator: secretAgent.GetTokenGenerator(o.WebhookSecretFile),
		helpProvider:   helpProviderGenerator(epa),
	}
	return server
}
