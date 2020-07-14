package http

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/sylr/cerberus/config"
	slackevents "github.com/sylr/cerberus/pkg/http/handlers/slack/events"
	"github.com/sylr/cerberus/pkg/slack"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// NewHTTPRouter returns an HTTP handler
func NewHTTPRouter(conf *config.Cerberus, safe *config.Safe) http.Handler {
	var subrouter *mux.Router
	var h http.Handler

	router := mux.NewRouter()

	// Slack client
	slackClient := slack.NewClient(&conf.Slack)

	// Profiling
	subrouter = router.PathPrefix("/debug/pprof/").Subrouter()
	subrouter.NewRoute().Handler(http.DefaultServeMux)

	// Metrics
	subrouter = router.Path("/metrics").Subrouter()
	subrouter.NewRoute().Handler(promhttp.Handler())

	// Slack events
	subrouter = router.PathPrefix("/slack/events").Subrouter()
	h = slackevents.NewHandler(conf, log.StandardLogger(), slackClient)
	subrouter.NewRoute().Handler(h)

	return router
}
