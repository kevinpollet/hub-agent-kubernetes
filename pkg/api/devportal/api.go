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

package devportal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	hubv1alpha1 "github.com/traefik/hub-agent-kubernetes/pkg/crd/api/hub/v1alpha1"
	logwrapper "github.com/traefik/hub-agent-kubernetes/pkg/logger"
	"github.com/traefik/hub-agent-kubernetes/pkg/platform"
)

const (
	headerHubGroups = "Hub-Groups"
	headerHubEmail  = "Hub-Email"
)

// Security schemes used to secure the exposed APIs.
const (
	securitySchemeQueryAuth  = "query_auth"
	securitySchemeBearerAuth = "bearer_auth"
)

// PortalAPI is a handler that exposes APIPortal information.
type PortalAPI struct {
	router     chi.Router
	httpClient *http.Client
	platform   PlatformClient

	portal *portal
}

// NewPortalAPI creates a new PortalAPI handler.
func NewPortalAPI(portal *portal, platformClient PlatformClient) (*PortalAPI, error) {
	client := retryablehttp.NewClient()
	client.RetryMax = 4
	client.Logger = logwrapper.NewRetryableHTTPWrapper(log.Logger.With().
		Str("component", "portal_api").
		Logger())

	p := &PortalAPI{
		router:     chi.NewRouter(),
		httpClient: client.StandardClient(),
		platform:   platformClient,
		portal:     portal,
	}

	p.router.Get("/apis", p.handleListAPIs)
	p.router.Get("/apis/{api}", p.handleGetAPISpec)
	p.router.Get("/collections/{collection}/apis/{api}", p.handleGetCollectionAPISpec)
	p.router.Get("/tokens", p.handleListTokens)
	p.router.Post("/tokens", p.handleCreateToken)
	p.router.Post("/tokens/suspend", p.handleSuspendToken)
	p.router.Delete("/tokens", p.handleDeleteToken)

	return p, nil
}

// ServeHTTP serves HTTP requests.
func (p *PortalAPI) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.router.ServeHTTP(rw, req)
}

func (p *PortalAPI) handleListTokens(rw http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("portal_name", p.portal.Name).Logger()

	userEmail := r.Header.Get(headerHubEmail)
	if userEmail == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	tokens, err := p.platform.ListUserTokens(r.Context(), userEmail)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to list user tokens")

		apiErr := platform.APIError{}
		if !errors.As(err, &apiErr) {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(apiErr.StatusCode)
		return
	}

	if tokens == nil {
		tokens = make([]platform.Token, 0)
	}

	rw.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(rw).Encode(tokens); err != nil {
		logger.Error().Err(err).Msg("Unable to list user tokens")
	}
}

type createTokenReq struct {
	Name string `json:"name"`
}
type createTokenResp struct {
	Token string `json:"token"`
}

func (p *PortalAPI) handleCreateToken(rw http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("portal_name", p.portal.Name).Logger()

	userEmail := r.Header.Get(headerHubEmail)
	if userEmail == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload createTokenReq
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Error().Err(err).Msg("Unable to decode payload")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := p.platform.CreateUserToken(r.Context(), userEmail, payload.Name)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to create user token")

		apiErr := platform.APIError{}
		if !errors.As(err, &apiErr) {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(apiErr.StatusCode)
		return
	}

	rw.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(rw).Encode(createTokenResp{Token: token}); err != nil {
		logger.Error().Err(err).Msg("Unable to create user token")
	}
}

type suspendTokenReq struct {
	Name    string `json:"name"`
	Suspend bool   `json:"suspend"`
}

func (p *PortalAPI) handleSuspendToken(rw http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("portal_name", p.portal.Name).Logger()

	userEmail := r.Header.Get(headerHubEmail)
	if userEmail == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload suspendTokenReq
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Error().Err(err).Msg("Unable to decode payload")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := p.platform.SuspendUserToken(r.Context(), userEmail, payload.Name, payload.Suspend); err != nil {
		logger.Error().Err(err).Msg("Unable to suspend user token")

		apiErr := platform.APIError{}
		if !errors.As(err, &apiErr) {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(apiErr.StatusCode)
	}

	rw.WriteHeader(http.StatusOK)
}

type deleteTokenReq struct {
	Name string `json:"name"`
}

func (p *PortalAPI) handleDeleteToken(rw http.ResponseWriter, r *http.Request) {
	logger := log.With().Str("portal_name", p.portal.Name).Logger()

	userEmail := r.Header.Get(headerHubEmail)
	if userEmail == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var payload deleteTokenReq
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logger.Error().Err(err).Msg("Unable to decode payload")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := p.platform.DeleteUserToken(r.Context(), userEmail, payload.Name); err != nil {
		logger.Error().Err(err).Msg("Unable to delete user token")

		apiErr := platform.APIError{}
		if !errors.As(err, &apiErr) {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(apiErr.StatusCode)
	}

	rw.WriteHeader(http.StatusNoContent)
}

func (p *PortalAPI) handleListAPIs(rw http.ResponseWriter, r *http.Request) {
	userGroups := r.Header.Values(headerHubGroups)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(rw).Encode(buildListResp(p.portal, userGroups)); err != nil {
		log.Error().Err(err).
			Str("portal_name", p.portal.Name).
			Msg("Write list APIs response")
	}
}

func (p *PortalAPI) handleGetAPISpec(rw http.ResponseWriter, r *http.Request) {
	apiNameNamespace := chi.URLParam(r, "api")

	logger := log.With().
		Str("portal_name", p.portal.Name).
		Str("api_name", apiNameNamespace).
		Logger()

	a, ok := p.portal.Gateway.APIs[apiNameNamespace]
	if !ok || !a.authorizes(r.Header.Values(headerHubGroups)) {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	p.serveAPISpec(logger.WithContext(r.Context()), rw, &p.portal.Gateway, nil, &a)
}

func (p *PortalAPI) handleGetCollectionAPISpec(rw http.ResponseWriter, r *http.Request) {
	collectionName := chi.URLParam(r, "collection")
	apiNameNamespace := chi.URLParam(r, "api")

	logger := log.With().
		Str("portal_name", p.portal.Name).
		Str("collection_name", collectionName).
		Str("api_name", apiNameNamespace).
		Logger()

	c, ok := p.portal.Gateway.Collections[collectionName]
	if !ok || !c.authorizes(r.Header.Values(headerHubGroups)) {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	a, ok := c.APIs[apiNameNamespace]
	if !ok {
		logger.Debug().Msg("API not found")
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	p.serveAPISpec(logger.WithContext(r.Context()), rw, &p.portal.Gateway, &c, &a)
}

func (p *PortalAPI) serveAPISpec(ctx context.Context, rw http.ResponseWriter, g *gateway, c *collection, a *api) {
	logger := log.Ctx(ctx)

	spec, err := p.getOpenAPISpec(ctx, &a.API)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to fetch OpenAPI spec")
		rw.WriteHeader(http.StatusBadGateway)

		return
	}

	var pathPrefix string
	if c != nil {
		pathPrefix = c.Spec.PathPrefix
	}
	pathPrefix = path.Join(pathPrefix, a.Spec.PathPrefix)

	// As soon as a CustomDomain is provided on the Gateway, the API is no longer accessible through the HubDomain.
	domains := g.Status.CustomDomains
	if len(domains) == 0 {
		domains = []string{g.Status.HubDomain}
	}

	if err = overrideServersAndSecurity(spec, domains, pathPrefix); err != nil {
		logger.Error().Err(err).Msg("Unable to adapt OpenAPI spec server and security configurations")
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(rw).Encode(spec); err != nil {
		logger.Error().Msg("Unable to serve OpenAPI spec")
	}
}

func (p *PortalAPI) getOpenAPISpec(ctx context.Context, a *hubv1alpha1.API) (*openapi3.T, error) {
	svc := a.Spec.Service

	var openapiURL *url.URL
	switch {
	case svc.OpenAPISpec.URL != "":
		u, err := url.Parse(svc.OpenAPISpec.URL)
		if err != nil {
			return nil, fmt.Errorf("parse OpenAPI URL %q: %w", svc.OpenAPISpec.URL, err)
		}
		openapiURL = u

	case svc.Port.Number != 0 || svc.OpenAPISpec.Port != nil && svc.OpenAPISpec.Port.Number != 0:
		protocol := svc.OpenAPISpec.Protocol
		if svc.OpenAPISpec.Protocol == "" {
			protocol = "http"
		}

		port := svc.Port.Number
		if svc.OpenAPISpec.Port != nil {
			port = svc.OpenAPISpec.Port.Number
		}

		namespace := a.Namespace
		if namespace == "" {
			namespace = "default"
		}

		openapiURL = &url.URL{
			Scheme: protocol,
			Host:   fmt.Sprint(svc.Name, ".", namespace, ":", port),
			Path:   svc.OpenAPISpec.Path,
		}
	default:
		return nil, errors.New("no spec endpoint specified")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openapiURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request %q: %w", openapiURL.String(), err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept", "application/yaml")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request %q: %w", openapiURL.String(), err)
	}

	rawSpec, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read spec %q: %w", openapiURL.String(), err)
	}

	// A new loader must be created each time. LoadFromData mutates the internal state of Loader.
	// LoadFromURI doesn't take a context, therefore, we must do the call ourselves.
	spec, err := openapi3.NewLoader().LoadFromData(rawSpec)
	if err != nil {
		return nil, fmt.Errorf("load OpenAPI spec: %w", err)
	}

	return spec, nil
}

func overrideServersAndSecurity(spec *openapi3.T, domains []string, pathPrefix string) error {
	if err := setServers(spec, domains, pathPrefix); err != nil {
		return fmt.Errorf("set servers: %w", err)
	}

	setSecurity(spec)
	clearSpecificServersAndSecurity(spec)
	return nil
}

func setServers(spec *openapi3.T, domains []string, pathPrefix string) error {
	var serverPath string
	if len(spec.Servers) > 0 && spec.Servers[0].URL != "" {
		// TODO: Handle variable substitutions before parsing the URL. (e.g. using Servers.BasePath)
		u, err := url.Parse(spec.Servers[0].URL)
		if err != nil {
			return fmt.Errorf("parse server URL %q: %w", spec.Servers[0].URL, err)
		}
		serverPath = u.Path
	}

	servers := make(openapi3.Servers, 0, len(domains))
	for _, domain := range domains {
		servers = append(servers, &openapi3.Server{
			URL: "https://" + domain + path.Join("/", pathPrefix, serverPath),
		})
	}
	spec.Servers = servers

	return nil
}

func setSecurity(spec *openapi3.T) {
	if spec.Components == nil {
		spec.Components = &openapi3.Components{}
	}

	spec.Components.SecuritySchemes = map[string]*openapi3.SecuritySchemeRef{
		securitySchemeQueryAuth: {
			Value: &openapi3.SecurityScheme{
				Type: "apiKey",
				In:   "query",
				Name: "api_key",
			},
		},
		securitySchemeBearerAuth: {
			Value: &openapi3.SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "opaque",
			},
		},
	}

	spec.Security = openapi3.SecurityRequirements{
		{
			securitySchemeQueryAuth:  make([]string, 0),
			securitySchemeBearerAuth: make([]string, 0),
		},
	}
}

func clearSpecificServersAndSecurity(spec *openapi3.T) {
	for _, path := range spec.Paths {
		if path == nil {
			continue
		}

		path.Servers = nil

		for _, op := range path.Operations() {
			if op == nil {
				continue
			}

			op.Servers = nil
			op.Security = nil
		}
	}
}

type listResp struct {
	Collections []collectionResp `json:"collections"`
	APIs        []apiResp        `json:"apis"`
}

type collectionResp struct {
	Name       string    `json:"name"`
	PathPrefix string    `json:"pathPrefix,omitempty"`
	APIs       []apiResp `json:"apis"`
}

type apiResp struct {
	Name       string `json:"name"`
	PathPrefix string `json:"pathPrefix"`
	SpecLink   string `json:"specLink"`
}

func buildListResp(p *portal, userGroups []string) listResp {
	var resp listResp
	for collectionName, c := range p.Gateway.Collections {
		if !c.authorizes(userGroups) {
			continue
		}

		cr := collectionResp{
			Name:       collectionName,
			PathPrefix: c.Spec.PathPrefix,
			APIs:       make([]apiResp, 0, len(c.APIs)),
		}

		for apiNameNamespace, a := range c.APIs {
			cr.APIs = append(cr.APIs, apiResp{
				Name:       a.Name,
				PathPrefix: path.Join(cr.PathPrefix, a.Spec.PathPrefix),
				SpecLink:   fmt.Sprintf("/collections/%s/apis/%s", collectionName, apiNameNamespace),
			})
		}
		sortAPIsResp(cr.APIs)

		resp.Collections = append(resp.Collections, cr)
	}
	sortCollectionsResp(resp.Collections)

	for apiNameNamespace, a := range p.Gateway.APIs {
		if !a.authorizes(userGroups) {
			continue
		}

		resp.APIs = append(resp.APIs, apiResp{
			Name:       a.Name,
			PathPrefix: a.Spec.PathPrefix,
			SpecLink:   fmt.Sprintf("/apis/%s", apiNameNamespace),
		})
	}
	sortAPIsResp(resp.APIs)

	if resp.APIs == nil {
		resp.APIs = make([]apiResp, 0)
	}
	if resp.Collections == nil {
		resp.Collections = make([]collectionResp, 0)
	}

	return resp
}

func sortAPIsResp(apis []apiResp) {
	sort.Slice(apis, func(i, j int) bool {
		return apis[i].Name < apis[j].Name
	})
}

func sortCollectionsResp(collections []collectionResp) {
	sort.Slice(collections, func(i, j int) bool {
		return collections[i].Name < collections[j].Name
	})
}
