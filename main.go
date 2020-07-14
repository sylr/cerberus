package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/sylr/cerberus/config"
	crbhttp "github.com/sylr/cerberus/pkg/http"
	"github.com/sylr/cerberus/pkg/http/handlers/safewrapper"

	"github.com/jessevdk/go-flags"
	"github.com/leebenson/conform"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	qdcache "github.com/sylr/go-libqd/cache"
	qdconfig "github.com/sylr/go-libqd/config"
)

var (
	version       = "v0.0.0"
	goVersion     = runtime.Version()
	configManager = qdconfig.GetManager(log.StandardLogger())
)

var (
	cerberusBuildInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "cerberus",
			Subsystem: "",
			Name:      "build_info",
			Help:      "Cerberus build info",
		},
		[]string{"version"},
	)
)

func init() {
	// Parse arguments to set the desirend log level as soon as possible
	conf := &config.Cerberus{}
	parser := flags.NewParser(conf, flags.Default)

	if _, err := parser.Parse(); err == nil {
		// Update logging level
		switch {
		case len(conf.Verbose) == 1:
			log.SetLevel(log.DebugLevel)
		case len(conf.Verbose) > 1:
			log.SetLevel(log.TraceLevel)
		default:
			log.SetLevel(log.InfoLevel)
		}
	} else {
		// Only log the info severity or above.
		log.SetLevel(log.InfoLevel)
	}

	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{
		DisableColors:  true,
		DisableSorting: false,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Set & Register build info metric
	cerberusBuildInfo.WithLabelValues(version).Set(1)
	prometheus.MustRegister(cerberusBuildInfo)

	// Conform
	redact := func(s string) string {
		if len(s) > 0 {
			return "<redacted>"
		}
		return ""
	}
	conform.AddSanitizer("redact", redact)
}

func main() {
	// looping for --version in args
	for _, val := range os.Args {
		if val == "--version" {
			fmt.Printf("cerberus version %s\n", version)
			os.Exit(0)
		} else if val == "--" {
			break
		}
	}

	// Configuration
	conf := &config.Cerberus{}
	safe := &config.Safe{Logger: log.StandardLogger()}
	ctx := context.Background()

	configManager.AddValidators(nil, safe.ListeningAddressValidator, safe.LogValidator)
	configManager.AddAppliers(nil, safe.LogApplier, safe.ReloadApplier)
	configManager.MakeConfig(ctx, nil, conf)

	if log.GetLevel() >= log.DebugLevel {
		confRedacted := conf.DeepCopy()
		conform.Strings(confRedacted)
		log.Debugf("Configuration %#v", confRedacted)
	}

	// HTTP router
	router := crbhttp.NewHTTPRouter(conf, safe)
	wrapper := safewrapper.New(router)

	// HTTP Server
	server := http.Server{
		Handler:      wrapper,
		Addr:         conf.ListeningAddress,
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go server.ListenAndServe()

	for i := 0; i < 100; i++ {
		qdcache.GetMeteredCache(2*time.Minute, 2*time.Minute).Set(fmt.Sprintf("%d", i), nil, 0)
	}

	// Replace router when new conf is sent through the config chan
	configChan := configManager.NewConfigChan(nil)
	for {
		select {
		case newConf := <-configChan:
			if log.GetLevel() >= log.DebugLevel {
				confRedacted := newConf.(*config.Cerberus).DeepCopy()
				conform.Strings(confRedacted)
				log.Debugf("Configuration %#v", confRedacted)
			}

			newRouter := crbhttp.NewHTTPRouter(newConf.(*config.Cerberus), safe)
			wrapper.SwapHandler(newRouter)
		}
	}
}
