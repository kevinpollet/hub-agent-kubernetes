/*
Copyright (C) 2022-2023 Traefik Labs

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/

package apikey

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/traefik/hub-agent-kubernetes/pkg/acp/expr"
	"golang.org/x/crypto/sha3"
)

// Config configures an API key ACP handler.
type Config struct {
	Header         string            `json:"header"`
	Query          string            `json:"query"`
	Cookie         string            `json:"cookie"`
	Keys           []Key             `json:"keys"`
	ForwardHeaders map[string]string `json:"forwardHeaders"`
}

// Key defines an API key.
type Key struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
	Value    string            `json:"value"`
}

type key struct {
	ID       string
	Metadata map[string]interface{}
}

// Handler is an API Key ACP Handler.
type Handler struct {
	name       string
	header     string
	query      string
	cookie     string
	keys       map[string]key
	fwdHeaders map[string]string
}

// NewHandler creates a new API key ACP Handler.
func NewHandler(cfg *Config, name string) (*Handler, error) {
	if cfg.Header == "" && cfg.Query == "" && cfg.Cookie == "" {
		return nil, errors.New("at least one of header, query or cookie is required")
	}

	keys := make(map[string]key)
	uniqIDs := make(map[string]struct{})
	for _, k := range cfg.Keys {
		if k.ID == "" || k.Value == "" {
			return nil, errors.New("empty ID or value")
		}

		if _, ok := uniqIDs[k.ID]; ok {
			return nil, fmt.Errorf("duplicated ID %q", k.ID)
		}
		uniqIDs[k.ID] = struct{}{}

		md := make(map[string]interface{}, len(k.Metadata)+1)
		for mk, mv := range k.Metadata {
			md[mk] = mv
		}
		// Key ID is not part of metadata, add is under the "_id" key.
		md["_id"] = k.ID

		keys[k.Value] = key{ID: k.ID, Metadata: md}
	}

	return &Handler{
		name:       name,
		header:     cfg.Header,
		query:      cfg.Query,
		cookie:     cfg.Cookie,
		keys:       keys,
		fwdHeaders: cfg.ForwardHeaders,
	}, nil
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	l := log.With().Str("handler_type", "APIKey").Str("handler_name", h.name).Logger()

	apiKey, err := h.getAPIkey(req)
	if err != nil {
		l.Debug().Err(err).Msg("Getting API key")
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	hash := make([]byte, 64)
	sha3.ShakeSum256(hash, []byte(apiKey))
	k, ok := h.keys[fmt.Sprintf("%x", hash)]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	if h.fwdHeaders != nil {
		hdrs, err := expr.PluckClaims(h.fwdHeaders, k.Metadata)
		if err != nil {
			l.Error().Err(err).Msg("Unable to set forwarded header")
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		for name, vals := range hdrs {
			for _, val := range vals {
				rw.Header().Add(name, val)
			}
		}
	}

	rw.WriteHeader(http.StatusOK)
}

// getAPIkey finds the API key from an HTTP request based on how the API key middleware was configured.
func (h *Handler) getAPIkey(req *http.Request) (string, error) {
	if h.header != "" {
		if hdr := req.Header.Get(h.header); hdr != "" {
			return hdr, nil
		}
	}

	if h.query != "" {
		if uri := originalURI(req.Header); uri != "" {
			parsedURI, err := url.Parse(uri)
			if err != nil {
				return "", err
			}

			if qry := parsedURI.Query().Get(h.query); qry != "" {
				return qry, nil
			}
		}
	}

	if h.cookie != "" {
		if cookie, _ := req.Cookie(h.cookie); cookie != nil && cookie.Value != "" {
			return cookie.Value, nil
		}
	}

	return "", errors.New("missing API key")
}

// originalURI gets the original URI that was sent to the ingress controller, regardless of its type.
// It currently supports Traefik (X-Forwarded-Uri) and Nginx Community (X-Original-Url).
func originalURI(hdr http.Header) string {
	if xfu := hdr.Get("X-Forwarded-Uri"); xfu != "" {
		return xfu
	}

	return hdr.Get("X-Original-Url")
}
