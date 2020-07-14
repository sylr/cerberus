package config

import (
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

func (c *Cerberus) ConfigFile() string {
	return c.File
}

type Slack struct {
	Token   string `yaml:"token" json:"token" toml:"tokeb" conform:"redact"`
	Verbose bool   `yaml:"verbose" json:"verbose" toml:"verbose" `
}

type CerberusMention struct {
	Messages []CerberusMentionMessage `yaml:"messages" json:"messages" toml:"messages"`
}

type CerberusMentionMessage struct {
	Text     string `yaml:"text" json:"text" toml:"text"`
	ImageURL string `yaml:"image_url" json:"image_url" toml:"image_url"`
}

type Safe struct {
	sync.RWMutex
	Logger *log.Logger
}

func (s *Safe) ListeningAddressValidator(currentConfig qdconfig.Config, newConfig qdconfig.Config) []error {
	conf := newConfig.(*Cerberus)

	if len(conf.ListeningAddress) == 0 {
		conf.ListeningAddress = "0.0.0.0:8080"
	}

	return nil
}

func (s *Safe) LogValidator(currentConfig qdconfig.Config, newConfig qdconfig.Config) []error {
	return nil
}

func (s *Safe) LogApplier(currentConfig qdconfig.Config, newConfig qdconfig.Config) error {
	conf := newConfig.(*Cerberus)

	switch {
	case len(conf.Verbose) == 1:
		log.SetLevel(log.DebugLevel)
	case len(conf.Verbose) > 1:
		log.SetLevel(log.TraceLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	return nil
}

func (s *Safe) ReloadApplier(currentConfig qdconfig.Config, newConfig qdconfig.Config) error {
	var currentConf *Cerberus
	var newConf *Cerberus

	s.Lock()
	defer s.Unlock()

	newConf = newConfig.(*Cerberus)

	if currentConfig != nil {
		currentConf = currentConfig.(*Cerberus)
		newConf.Reloads = currentConf.Reloads + 1

		metricConfigReloadsTotal.WithLabelValues().Inc()
	} else {
		newConf.Reloads = 0
	}

	return nil
}
