package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/http/pprof"
	"net/url"

	"github.com/rs/zerolog/log"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Route struct {
	Url       string
	MatchWord string
}

// WithHealthz sets up and adds health check endpoints (/healthz and /debug/pprof/)
// and metrics endpoint (/metrics) to a new router
func (a *App) WithHealthz() *App {
	i := mux.NewRouter()
	a.healthy = true
	i.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if a.healthy {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ok"))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Not Ok"))
		}
	})
	i.HandleFunc("/debug/pprof/", pprof.Index)
	i.Handle("/metrics", promhttp.Handler())
	a.i = i
	return a
}

// WithRoutes initializes a new router, sets up logging middleware, and assigns
// the router to the App's router field, returning the updated App.
func (a *App) WithRoutes() *App {
	e := mux.NewRouter()
	e.Use(a.loggingMiddleware)
	e.SkipClean(true)
	a.e = e
	a.WithLoki()
	a.WithThanos()
	return a
}

// WithLoki configures and adds a set of Loki API routes to the App's router,
// logging warnings if the Loki URL is not set, and returns the updated App.
//
// Routes are based on Loki HTTP API Query Endpoints:
// https://grafana.com/docs/loki/latest/reference/loki-http-api/#query-endpoints
func (a *App) WithLoki() *App {
	if a.Cfg.Loki.URL == "" {
		log.Warn().Msg("Loki URL not set, skipping Loki routes")
		return a
	}
	routes := []Route{
		// Query Endpoints - https://grafana.com/docs/loki/latest/reference/loki-http-api/#instant-queries
		{Url: "/api/v1/query", MatchWord: "query"},
		// Range Queries - https://grafana.com/docs/loki/latest/reference/loki-http-api/#range-queries
		{Url: "/api/v1/query_range", MatchWord: "query"},
		// Labels - https://grafana.com/docs/loki/latest/reference/loki-http-api/#query-labels
		{Url: "/api/v1/labels", MatchWord: "query"},
		// Label Values - https://grafana.com/docs/loki/latest/reference/loki-http-api/#query-label-values
		{Url: "/api/v1/label/{label}/values", MatchWord: "query"},
		// Series - https://grafana.com/docs/loki/latest/reference/loki-http-api/#series
		{Url: "/api/v1/series", MatchWord: "match[]"},
		// Index Stats - https://grafana.com/docs/loki/latest/reference/loki-http-api/#statistics
		{Url: "/api/v1/index/stats", MatchWord: "query"},
		// Index Volume - https://grafana.com/docs/loki/latest/reference/loki-http-api/#volume
		{Url: "/api/v1/index/volume", MatchWord: "query"},
		// Index Volume Range - https://grafana.com/docs/loki/latest/reference/loki-http-api/#volume-range
		{Url: "/api/v1/index/volume_range", MatchWord: "query"},
		// Patterns - https://grafana.com/docs/loki/latest/reference/loki-http-api/#detected-patterns
		{Url: "/api/v1/patterns", MatchWord: "query"},
		// Tail - https://grafana.com/docs/loki/latest/reference/loki-http-api/#stream-logs
		{Url: "/api/v1/tail", MatchWord: "query"},
		// Additional Loki endpoints (not query endpoints)
		// Format Query - https://grafana.com/docs/loki/latest/reference/loki-http-api/#format-a-logql-query
		{Url: "/api/v1/format_query", MatchWord: "query"},
		// Build Info - https://grafana.com/docs/loki/latest/reference/loki-http-api/#show-build-information
		{Url: "/api/v1/status/buildinfo", MatchWord: "query"},
		// Query Exemplars - Prometheus endpoint (https://prometheus.io/docs/prometheus/latest/querying/api/#querying-exemplars)
		// Note: This is a Prometheus/Thanos endpoint, not a Loki endpoint, but included for compatibility
		{Url: "/api/v1/query_exemplars", MatchWord: "query"},
	}
	lokiRouter := a.e.PathPrefix("/loki").Subrouter()
	for _, route := range routes {
		log.Trace().Any("route", route).Msg("Loki route")
		lokiRouter.HandleFunc(route.Url, handler(route.MatchWord,
			LogQLEnforcer(struct{}{}),
			a.Cfg.Loki.TenantLabel,
			a.Cfg.Loki.URL,
			a.Cfg.Loki.UseMutualTLS,
			a.Cfg.Loki.Headers,
			a)).Name(route.Url)
	}
	return a
}

// WithThanos configures and adds a set of Thanos API routes to the App's router,
// logging warnings if the Thanos URL is not set, and returns the updated App.
//
// Routes are based on Prometheus HTTP API:
// https://prometheus.io/docs/prometheus/latest/querying/api/
func (a *App) WithThanos() *App {
	if a.Cfg.Thanos.URL == "" {
		log.Warn().Msg("Thanos URL not set, skipping Thanos routes")
		return a
	}
	routes := []Route{
		// Query Endpoints - https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
		{Url: "/api/v1/query", MatchWord: "query"},
		// Range Queries - https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries
		{Url: "/api/v1/query_range", MatchWord: "query"},
		// Format Query - https://prometheus.io/docs/prometheus/latest/querying/api/#formatting-query-expressions
		{Url: "/api/v1/format_query", MatchWord: "query"},
		// Metadata Endpoints
		// Series - https://prometheus.io/docs/prometheus/latest/querying/api/#finding-series-by-label-matchers
		{Url: "/api/v1/series", MatchWord: "match[]"},
		// Labels - https://prometheus.io/docs/prometheus/latest/querying/api/#getting-label-names
		{Url: "/api/v1/labels", MatchWord: "match[]"},
		// Label Values - https://prometheus.io/docs/prometheus/latest/querying/api/#querying-label-values
		{Url: "/api/v1/label/{label}/values", MatchWord: "match[]"},
		// Query Exemplars - https://prometheus.io/docs/prometheus/latest/querying/api/#querying-exemplars
		{Url: "/api/v1/query_exemplars", MatchWord: "query"},
		// Metadata - https://prometheus.io/docs/prometheus/latest/querying/api/#querying-metric-metadata
		{Url: "/api/v1/metadata", MatchWord: "query"},
		// Status Endpoints
		// Build Info - https://prometheus.io/docs/prometheus/latest/querying/api/#build-information
		{Url: "/api/v1/status/buildinfo", MatchWord: "query"},
		// Non-Prometheus endpoints (Thanos or compatibility)
		// Note: These endpoints are not part of standard Prometheus API
		{Url: "/api/v1/tail", MatchWord: "query"},
		{Url: "/api/v1/index/stats", MatchWord: "query"},
	}
	thanosRouter := a.e.PathPrefix("").Subrouter()
	for _, route := range routes {
		log.Trace().Any("route", route).Msg("Thanos route")
		thanosRouter.HandleFunc(route.Url,
			handler(route.MatchWord,
				PromQLEnforcer(struct{}{}),
				a.Cfg.Thanos.TenantLabel,
				a.Cfg.Thanos.URL,
				a.Cfg.Thanos.UseMutualTLS,
				a.Cfg.Thanos.Headers,
				a)).Name(route.Url)

	}
	return a
}

// handler function orchestrates the request flow through the proxy, comprising
// authentication, conditional enforcement, and forwarding to the upstream server.
//
// Initially, it retrieves the OAuth token and validates it.
//
// Subsequently, it validates labels retrieved from the token and determines whether
// enforcement should be skipped based on them. If an error occurs during label
// validation, it is logged and a forbidden status response is dispatched. If enforcement
// is opted to be skipped, the request is streamed directly to the upstream server without
// further checks.
//
// If the flow doesn’t skip enforcement, the function enforces the request based on the
// provided labels and other relevant parameters. Should any enforcement error arise, it is
// logged and a forbidden status is sent to the client.
//
// Finally, if all checks and possible enforcement pass successfully, the request is
// streamed to the upstream server.
func handler(matchWord string, enforcer EnforceQL, tl string, dsURL string, tls bool, headers map[string]string, a *App) func(http.ResponseWriter, *http.Request) {
	upstreamURL, err := url.Parse(dsURL)
	if err != nil {
		log.Fatal().Err(err).Str("url", dsURL).Msg("Error parsing URL")
	}
	return func(w http.ResponseWriter, r *http.Request) {
		oauthToken, err := getToken(r, a)
		if err != nil {
			logAndWriteError(w, http.StatusForbidden, err, "")
		}

		labels, skip, err := validateLabels(oauthToken, a)
		if err != nil {
			logAndWriteError(w, http.StatusForbidden, err, "")
			return
		}
		if skip {
			streamUp(w, r, upstreamURL, tls, headers, a)
			return
		}

		err = enforceRequest(r, enforcer, labels, tl, matchWord)
		if err != nil {
			logAndWriteError(w, http.StatusForbidden, err, "")
			return
		}

		if _, ok := enforcer.(LogQLEnforcer); ok {
			err := setActorHeaderLogQL(r, oauthToken, a)
			if err != nil {
				logAndWriteError(w, http.StatusForbidden, err, "")
				return
			}
		}
		if _, ok := enforcer.(PromQLEnforcer); ok {
			err := setActorHeaderPromQL(r, oauthToken, a)
			if err != nil {
				logAndWriteError(w, http.StatusForbidden, err, "")
				return
			}
		}

		streamUp(w, r, upstreamURL, tls, headers, a)
	}
}

func setActorHeaderLogQL(r *http.Request, token OAuthToken, a *App) error {
	if a.Cfg.Loki.ActorHeader != "" {
		data := fmt.Sprintf("%s%s", token.PreferredUsername, token.Email)
		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		r.Header.Set(a.Cfg.Loki.ActorHeader, encoded)
	}
	return nil
}

func setActorHeaderPromQL(r *http.Request, token OAuthToken, a *App) error {
	if a.Cfg.Thanos.ActorHeader != "" {
		data := fmt.Sprintf("%s%s", token.PreferredUsername, token.Email)
		encoded := base64.StdEncoding.EncodeToString([]byte(data))
		r.Header.Set(a.Cfg.Thanos.ActorHeader, encoded)
	}
	return nil
}

// streamUp forwards the provided HTTP request to the specified upstream URL using
// a reverse proxy.It serves the upstream content back to the original client.
func streamUp(w http.ResponseWriter, r *http.Request, upstreamURL *url.URL, tls bool, headers map[string]string, a *App) {
	setHeaders(r, tls, headers, a.ServiceAccountToken)
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	proxy.ServeHTTP(w, r)
}

// setHeaders modifies the HTTP request headers to set the Authorization and
// other headers based on the provided arguments.
func setHeaders(r *http.Request, tls bool, header map[string]string, sat string) {
	if !tls {
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", sat))
	}
	for k, v := range header {
		r.Header.Set(k, v)
	}
}
