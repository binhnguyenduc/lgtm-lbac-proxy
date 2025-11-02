package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

// EnforceQL represents an interface that any query language enforcement should implement.
// It contains methods to enforce queries based on tenant labels and label match.
type EnforceQL interface {
	// Enforce enforces query with simple label value matching (backward compatible)
	Enforce(query string, tenantLabels map[string]bool, labelMatch string) (string, error)

	// EnforceMulti enforces query with flexible multi-label policy
	EnforceMulti(query string, policy LabelPolicy) (string, error)
}

// enforceRequest enforces the incoming HTTP request based on its method (GET or POST).
// It delegates the enforcement to enforceGet or enforcePost functions based on the HTTP method of the request.
func enforceRequest(r *http.Request, enforce EnforceQL, tenantLabels map[string]bool, labelMatch string, queryMatch string) error {
	switch r.Method {
	case http.MethodGet:
		return enforceGet(r, enforce, tenantLabels, labelMatch, queryMatch)
	case http.MethodPost:
		return enforcePost(r, enforce, tenantLabels, labelMatch, queryMatch)
	default:
		return fmt.Errorf("invalid method")
	}
}

// enforceGet enforces the query parameters of the incoming GET HTTP request.
// It modifies the request URL's query parameters to ensure they adhere to tenant labels and label match.
func enforceGet(r *http.Request, enforce EnforceQL, tenantLabels map[string]bool, labelMatch string, queryMatch string) error {
	log.Trace().Str("kind", "urlmatch").Str("queryMatch", queryMatch).Str("query", r.URL.Query().Get("query")).Str("match[]", r.URL.Query().Get("match[]")).Msg("")

	query, err := enforce.Enforce(r.URL.Query().Get(queryMatch), tenantLabels, labelMatch)
	if err != nil {
		return err
	}
	log.Trace().Any("url", r.URL).Msg("pre enforced url")
	values := r.URL.Query()
	values.Set(queryMatch, query)
	r.URL.RawQuery = values.Encode()
	log.Trace().Any("url", r.URL).Msg("post enforced url")

	r.Body = io.NopCloser(strings.NewReader(""))
	r.ContentLength = 0
	return nil
}

// enforcePost enforces the form values of the incoming POST HTTP request.
// It modifies the request's form values to ensure they adhere to tenant labels and label match.
func enforcePost(r *http.Request, enforce EnforceQL, tenantLabels map[string]bool, labelMatch string, queryMatch string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	log.Trace().Str("kind", "bodymatch").Str("queryMatch", queryMatch).Str("query", r.PostForm.Get("query")).Str("match[]", r.PostForm.Get("match[]")).Msg("")

	query := r.PostForm.Get(queryMatch)
	query, err := enforce.Enforce(query, tenantLabels, labelMatch)
	if err != nil {
		return err
	}

	_ = r.Body.Close()
	r.PostForm.Set(queryMatch, query)
	newBody := r.PostForm.Encode()
	r.Body = io.NopCloser(strings.NewReader(newBody))
	r.ContentLength = int64(len(newBody))
	r.URL.RawQuery = ""
	return nil
}

// enforceRequestWithPolicy enforces the incoming HTTP request using LabelPolicy.
// It provides multi-label enforcement with flexible operators and logic.
func enforceRequestWithPolicy(r *http.Request, enforce EnforceQL, policy *LabelPolicy, queryMatch string) error {
	switch r.Method {
	case http.MethodGet:
		return enforceGetWithPolicy(r, enforce, *policy, queryMatch)
	case http.MethodPost:
		return enforcePostWithPolicy(r, enforce, *policy, queryMatch)
	default:
		return fmt.Errorf("invalid method")
	}
}

// enforceGetWithPolicy enforces the query parameters of the incoming GET HTTP request using LabelPolicy.
// It modifies the request URL's query parameters to ensure they adhere to the label policy.
func enforceGetWithPolicy(r *http.Request, enforce EnforceQL, policy LabelPolicy, queryMatch string) error {
	log.Trace().Str("kind", "urlmatch").Str("queryMatch", queryMatch).Str("query", r.URL.Query().Get("query")).Msg("enforcing with policy")

	query, err := enforce.EnforceMulti(r.URL.Query().Get(queryMatch), policy)
	if err != nil {
		return err
	}
	log.Trace().Any("url", r.URL).Msg("pre enforced url")
	values := r.URL.Query()
	values.Set(queryMatch, query)
	r.URL.RawQuery = values.Encode()
	log.Trace().Any("url", r.URL).Msg("post enforced url")

	r.Body = io.NopCloser(strings.NewReader(""))
	r.ContentLength = 0
	return nil
}

// enforcePostWithPolicy enforces the form values of the incoming POST HTTP request using LabelPolicy.
// It modifies the request's form values to ensure they adhere to the label policy.
func enforcePostWithPolicy(r *http.Request, enforce EnforceQL, policy LabelPolicy, queryMatch string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	log.Trace().Str("kind", "bodymatch").Str("queryMatch", queryMatch).Str("query", r.PostForm.Get("query")).Msg("enforcing with policy")

	query := r.PostForm.Get(queryMatch)
	query, err := enforce.EnforceMulti(query, policy)
	if err != nil {
		return err
	}

	_ = r.Body.Close()
	r.PostForm.Set(queryMatch, query)
	newBody := r.PostForm.Encode()
	r.Body = io.NopCloser(strings.NewReader(newBody))
	r.ContentLength = int64(len(newBody))
	r.URL.RawQuery = ""
	return nil
}
