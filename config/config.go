package config

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	qdconfig "github.com/sylr/go-libqd/config"
)

//go:generate deepcopy-gen --input-dirs . --output-package . --output-file-base config_deepcopy --go-header-file /dev/null
//+k8s:deepcopy-gen=true
//+k8s:deepcopy-gen:interfaces=github.com/sylr/go-libqd/config.Config

var (
	metricConfigReloadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cerberus",
			Subsystem: "config",
			Name:      "reloads_total",
			Help:      "Number of config reloads",
		},
		[]string{},
	)
)

// Cerberus implements github.com/sylr/go-libqd/config.Config
type Cerberus struct {
	Reloads          int64            `yaml:"-"`
	Version          bool             `                                                       long:"version"`
	File             string           `                                             short:"f" long:"config"`
	Verbose          []bool           `yaml:"verbose" json:"verbose" toml:"verbose" short:"v" long:"verbose"`
	ListeningAddress string           `yaml:"address" json:"address" toml:"address" short:"a" long:"address"`
	Slack            Slack            `yaml:"slack"`
	CerberusMention  *CerberusMention `yaml:"cerberus_mention" json:"cerberus_mention" toml:"cerberus_mention"`
}

// ConfigFile ...
func (c *Cerberus) ConfigFile() string {
	return c.File
}

// Slack ...
type Slack struct {
	Token   string `yaml:"token" json:"token" toml:"tokeb" conform:"redact"`
	Verbose bool   `yaml:"verbose" json:"verbose" toml:"verbose" `
}

// CerberusMention ...
type CerberusMention struct {
	Messages []CerberusMentionMessage `yaml:"messages" json:"messages" toml:"messages"`
}

// CerberusMentionMessage ...
type CerberusMentionMessage struct {
	Text     string `yaml:"text" json:"text" toml:"text"`
	ImageURL string `yaml:"image_url" json:"image_url" toml:"image_url"`
}

// Safe is a struct Validators and Appliers.
type Safe struct {
	Logger *log.Logger
	mu     sync.Mutex
}

// ListeningAddressValidator defaults the listening address to "0.0.0.0:8080" if not set.
func (s *Safe) ListeningAddressValidator(currentConfig qdconfig.Config, newConfig qdconfig.Config) []error {
	var errors []error
	newConf := newConfig.(*Cerberus)

	if len(newConf.ListeningAddress) == 0 {
		newConf.ListeningAddress = "0.0.0.0:8080"
	}

	if currentConfig != nil {
		curConf := currentConfig.(*Cerberus)

		if curConf.ListeningAddress != newConf.ListeningAddress {
			errors = append(errors, fmt.Errorf("Changing listening address is not implemented"))
		}
	}

	return errors
}

// LogValidator does nothing
func (s *Safe) LogValidator(currentConfig qdconfig.Config, newConfig qdconfig.Config) []error {
	return nil
}

// LogApplier sets the log level
func (s *Safe) LogApplier(currentConfig qdconfig.Config, newConfig qdconfig.Config) error {
	conf := newConfig.(*Cerberus)

	switch {
	case len(conf.Verbose) == 1:
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
	case len(conf.Verbose) > 1:
		log.SetLevel(log.TraceLevel)
		log.SetReportCaller(true)
	default:
		log.SetLevel(log.InfoLevel)
		log.SetReportCaller(false)
	}

	return nil
}

// ReloadApplier incerments newConfig.Reloads and reload prometheus metric
func (s *Safe) ReloadApplier(currentConfig qdconfig.Config, newConfig qdconfig.Config) error {
	var currentConf, newConf *Cerberus

	newConf = newConfig.(*Cerberus)

	if currentConfig != nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		currentConf = currentConfig.(*Cerberus)
		newConf.Reloads = currentConf.Reloads + 1

		metricConfigReloadsTotal.WithLabelValues().Inc()
	}

	return nil
}
