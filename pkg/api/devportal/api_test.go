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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hubv1alpha1 "github.com/traefik/hub-agent-kubernetes/pkg/crd/api/hub/v1alpha1"
	"github.com/traefik/hub-agent-kubernetes/pkg/platform"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testEmail     = "john.doe@example.com"
	testTokenName = "my-token"
)

var testPortal = portal{
	APIPortal: hubv1alpha1.APIPortal{ObjectMeta: metav1.ObjectMeta{Name: "my-portal"}},
	Gateway: gateway{
		APIGateway: hubv1alpha1.APIGateway{
			ObjectMeta: metav1.ObjectMeta{Name: "my-gateway"},
			Status: hubv1alpha1.APIGatewayStatus{
				HubDomain: "majestic-beaver-123.hub-traefik.io",
				CustomDomains: []string{
					"api.my-company.example.com",
				},
			},
		},
		Collections: map[string]collection{
			"products": {
				APICollection: hubv1alpha1.APICollection{
					ObjectMeta: metav1.ObjectMeta{Name: "products"},
					Spec: hubv1alpha1.APICollectionSpec{
						PathPrefix: "/products",
					},
				},
				APIs: map[string]api{
					"books@products-ns": {
						API: hubv1alpha1.API{
							ObjectMeta: metav1.ObjectMeta{Name: "books", Namespace: "products-ns"},
							Spec: hubv1alpha1.APISpec{
								PathPrefix: "/books",
								Service: hubv1alpha1.APIService{
									Name: "books-svc",
									Port: hubv1alpha1.APIServiceBackendPort{Number: 80},
									OpenAPISpec: hubv1alpha1.OpenAPISpec{
										URL: "http://my-oas-registry.example.com/artifacts/12345",
									},
								},
							},
						},
						authorizedGroups: []string{"supplier"},
					},
					"groceries@products-ns": {
						API: hubv1alpha1.API{
							ObjectMeta: metav1.ObjectMeta{Name: "groceries", Namespace: "products-ns"},
							Spec: hubv1alpha1.APISpec{
								PathPrefix: "/groceries",
								Service: hubv1alpha1.APIService{
									Name:        "groceries-svc",
									Port:        hubv1alpha1.APIServiceBackendPort{Number: 8080},
									OpenAPISpec: hubv1alpha1.OpenAPISpec{Path: "/spec.json"},
								},
							},
						},
						authorizedGroups: []string{"supplier"},
					},
					"furnitures@products-ns": {
						API: hubv1alpha1.API{
							ObjectMeta: metav1.ObjectMeta{Name: "furnitures", Namespace: "products-ns"},
							Spec: hubv1alpha1.APISpec{
								PathPrefix: "/furnitures",
								Service: hubv1alpha1.APIService{
									Name: "furnitures-svc",
									Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
									OpenAPISpec: hubv1alpha1.OpenAPISpec{
										Path: "/spec.json",
										Port: &hubv1alpha1.APIServiceBackendPort{
											Number: 9000,
										},
									},
								},
							},
						},
						authorizedGroups: []string{"supplier"},
					},
					"toys@products-ns": {
						API: hubv1alpha1.API{
							ObjectMeta: metav1.ObjectMeta{Name: "toys", Namespace: "products-ns"},
							Spec: hubv1alpha1.APISpec{
								PathPrefix: "/toys",
								Service: hubv1alpha1.APIService{
									Name: "toys-svc",
									Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
								},
							},
						},
						authorizedGroups: []string{"supplier"},
					},
				},
				authorizedGroups: []string{"supplier"},
			},
		},
		APIs: map[string]api{
			"managers@people-ns": {
				API: hubv1alpha1.API{
					ObjectMeta: metav1.ObjectMeta{Name: "managers", Namespace: "people-ns"},
					Spec: hubv1alpha1.APISpec{
						PathPrefix: "/managers",
						Service: hubv1alpha1.APIService{
							Name: "managers-svc",
							Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
							OpenAPISpec: hubv1alpha1.OpenAPISpec{
								URL: "http://my-oas-registry.example.com/artifacts/456",
							},
						},
					},
				},
				authorizedGroups: []string{"supplier"},
			},
			"notifications@default": {
				API: hubv1alpha1.API{
					ObjectMeta: metav1.ObjectMeta{Name: "notifications", Namespace: "default"},
					Spec: hubv1alpha1.APISpec{
						PathPrefix: "/notifications",
						Service: hubv1alpha1.APIService{
							Name: "notifications-svc",
							Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
							OpenAPISpec: hubv1alpha1.OpenAPISpec{
								Path: "/spec.json",
							},
						},
					},
				},
				authorizedGroups: []string{"supplier"},
			},
			"metrics@default": {
				API: hubv1alpha1.API{
					ObjectMeta: metav1.ObjectMeta{Name: "metrics", Namespace: "default"},
					Spec: hubv1alpha1.APISpec{
						PathPrefix: "/metrics",
						Service: hubv1alpha1.APIService{
							Name: "metrics-svc",
							Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
							OpenAPISpec: hubv1alpha1.OpenAPISpec{
								Path: "/spec.json",
								Port: &hubv1alpha1.APIServiceBackendPort{
									Number: 9000,
								},
							},
						},
					},
				},
				authorizedGroups: []string{"supplier"},
			},
			"health@default": {
				API: hubv1alpha1.API{
					ObjectMeta: metav1.ObjectMeta{Name: "health", Namespace: "default"},
					Spec: hubv1alpha1.APISpec{
						PathPrefix: "/health",
						Service: hubv1alpha1.APIService{
							Name: "health-svc",
							Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
						},
					},
				},
				authorizedGroups: []string{"supplier"},
			},
			"api@default": {
				API: hubv1alpha1.API{
					ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "default"},
					Spec: hubv1alpha1.APISpec{
						PathPrefix: "/api",
						Service: hubv1alpha1.APIService{
							Name: "api-svc",
							Port: hubv1alpha1.APIServiceBackendPort{Number: 8080},
						},
					},
				},
				authorizedGroups: []string{"developer"},
			},
		},
	},
}

func TestPortalAPI_Router_listTokens(t *testing.T) {
	tests := []struct {
		desc           string
		tokens         []platform.Token
		platformErr    error
		wantStatusCode int
	}{
		{
			desc: "list tokens",
			tokens: []platform.Token{
				{Name: "token-1", Suspended: false},
				{Name: "token-2", Suspended: true},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			desc: "not found",
			platformErr: platform.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "conflict",
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			desc:           "unexpected platform error",
			platformErr:    errors.New("boom"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			platformClient := newPlatformClientMock(t)
			platformClient.OnListUserTokens(testEmail).TypedReturns(test.tokens, test.platformErr)

			a, err := NewPortalAPI(&testPortal, platformClient)
			require.NoError(t, err)

			srv := httptest.NewServer(a)

			req, err := http.NewRequest(http.MethodGet, srv.URL+"/tokens", http.NoBody)
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, test.wantStatusCode, resp.StatusCode)
			if test.wantStatusCode == http.StatusOK {
				var got []platform.Token
				err = json.NewDecoder(resp.Body).Decode(&got)
				require.NoError(t, err)

				assert.Equal(t, test.tokens, got)
			}
		})
	}
}

func TestPortalAPI_Router_createToken(t *testing.T) {
	tests := []struct {
		desc           string
		token          string
		platformErr    error
		wantStatusCode int
	}{
		{
			desc:           "created",
			token:          "token",
			wantStatusCode: http.StatusCreated,
		},
		{
			desc: "conflict",
			platformErr: platform.APIError{
				StatusCode: http.StatusConflict,
				Message:    "conflict",
			},
			wantStatusCode: http.StatusConflict,
		},
		{
			desc: "not found",
			platformErr: platform.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "conflict",
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			desc:           "unexpected platform error",
			platformErr:    errors.New("boom"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			platformClient := newPlatformClientMock(t)
			platformClient.OnCreateUserToken(testEmail, testTokenName).TypedReturns(test.token, test.platformErr)

			a, err := NewPortalAPI(&testPortal, platformClient)
			require.NoError(t, err)

			srv := httptest.NewServer(a)

			body, err := json.Marshal(createTokenReq{Name: testTokenName})
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/tokens", bytes.NewReader(body))
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, test.wantStatusCode, resp.StatusCode)

			if test.token == "" {
				return
			}

			var got createTokenResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
			assert.Equal(t, createTokenResp{Token: test.token}, got)
		})
	}
}

func TestPortalAPI_Router_suspendToken(t *testing.T) {
	tests := []struct {
		desc           string
		suspend        bool
		platformErr    error
		wantStatusCode int
	}{
		{
			desc:           "suspended",
			suspend:        true,
			wantStatusCode: http.StatusOK,
		},
		{
			desc:           "un-suspended",
			suspend:        false,
			wantStatusCode: http.StatusOK,
		},
		{
			desc: "not found",
			platformErr: platform.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "conflict",
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			desc:           "unexpected platform error",
			platformErr:    errors.New("boom"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			platformClient := newPlatformClientMock(t)
			platformClient.OnSuspendUserToken(testEmail, testTokenName, test.suspend).TypedReturns(test.platformErr)

			a, err := NewPortalAPI(&testPortal, platformClient)
			require.NoError(t, err)

			srv := httptest.NewServer(a)

			body, err := json.Marshal(suspendTokenReq{Name: testTokenName, Suspend: test.suspend})
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodPost, srv.URL+"/tokens/suspend", bytes.NewReader(body))
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, test.wantStatusCode, resp.StatusCode)
		})
	}
}

func TestPortalAPI_Router_deleteToken(t *testing.T) {
	tests := []struct {
		desc           string
		platformErr    error
		wantStatusCode int
	}{
		{
			desc:           "deleted",
			wantStatusCode: http.StatusNoContent,
		},
		{
			desc: "not found",
			platformErr: platform.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "conflict",
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			desc:           "unexpected platform error",
			platformErr:    errors.New("boom"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			platformClient := newPlatformClientMock(t)
			platformClient.OnDeleteUserToken(testEmail, testTokenName).TypedReturns(test.platformErr)

			a, err := NewPortalAPI(&testPortal, platformClient)
			require.NoError(t, err)

			srv := httptest.NewServer(a)

			body, err := json.Marshal(deleteTokenReq{Name: testTokenName})
			require.NoError(t, err)

			req, err := http.NewRequest(http.MethodDelete, srv.URL+"/tokens", bytes.NewReader(body))
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, test.wantStatusCode, resp.StatusCode)
		})
	}
}

func TestPortalAPI_Router_listAPIs(t *testing.T) {
	a, err := NewPortalAPI(&testPortal, nil)
	require.NoError(t, err)

	srv := httptest.NewServer(a)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/apis", http.NoBody)
	require.NoError(t, err)

	req.Header.Add("Hub-Email", testEmail)
	req.Header.Add("Hub-Groups", "supplier")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var got listResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, listResp{
		Collections: []collectionResp{
			{
				Name:       "products",
				PathPrefix: "/products",
				APIs: []apiResp{
					{Name: "books", PathPrefix: "/products/books", SpecLink: "/collections/products/apis/books@products-ns"},
					{Name: "furnitures", PathPrefix: "/products/furnitures", SpecLink: "/collections/products/apis/furnitures@products-ns"},
					{Name: "groceries", PathPrefix: "/products/groceries", SpecLink: "/collections/products/apis/groceries@products-ns"},
					{Name: "toys", PathPrefix: "/products/toys", SpecLink: "/collections/products/apis/toys@products-ns"},
				},
			},
		},
		APIs: []apiResp{
			{Name: "health", PathPrefix: "/health", SpecLink: "/apis/health@default"},
			{Name: "managers", PathPrefix: "/managers", SpecLink: "/apis/managers@people-ns"},
			{Name: "metrics", PathPrefix: "/metrics", SpecLink: "/apis/metrics@default"},
			{Name: "notifications", PathPrefix: "/notifications", SpecLink: "/apis/notifications@default"},
		},
	}, got)
}

func TestPortalAPI_Router_listAPIs_noAPIsAndCollections(t *testing.T) {
	var p portal
	a, err := NewPortalAPI(&p, nil)
	require.NoError(t, err)

	srv := httptest.NewServer(a)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/apis", http.NoBody)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"collections": [],
		"apis": []
	}`, string(got))
}

func TestPortalAPI_Router_getCollectionAPISpec(t *testing.T) {
	tests := []struct {
		desc       string
		collection string
		api        string
		wantURL    string
		statusCode int
	}{
		{
			desc:       "OpenAPI spec defined using an external URL",
			collection: "products",
			api:        "books@products-ns",
			wantURL:    "http://my-oas-registry.example.com/artifacts/12345",
			statusCode: http.StatusOK,
		},
		{
			desc:       "OpenAPI spec defined with a path on the service",
			collection: "products",
			api:        "groceries@products-ns",
			wantURL:    "http://groceries-svc.products-ns:8080/spec.json",
			statusCode: http.StatusOK,
		},
		{
			desc:       "OpenAPI spec defined with a path on the service and a specific port",
			collection: "products",
			api:        "furnitures@products-ns",
			wantURL:    "http://furnitures-svc.products-ns:9000/spec.json",
			statusCode: http.StatusOK,
		},
		{
			desc:       "No OpenAPI spec defined",
			collection: "products",
			api:        "toys@products-ns",
			wantURL:    "http://toys-svc.products-ns:8080/",
			statusCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			svcSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if r.URL.String() != test.wantURL {
					t.Logf("expected URL %q got %q", test.wantURL, r.URL.String())
					rw.WriteHeader(http.StatusNotFound)
					return
				}

				if err := json.NewEncoder(rw).Encode(openapi3.T{OpenAPI: "v3.0"}); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
				}
			}))

			a, err := NewPortalAPI(&testPortal, nil)
			require.NoError(t, err)
			a.httpClient = buildProxyClient(t, svcSrv.URL)

			apiSrv := httptest.NewServer(a)

			uri := fmt.Sprintf("%s/collections/%s/apis/%s", apiSrv.URL, test.collection, test.api)
			req, err := http.NewRequest(http.MethodGet, uri, http.NoBody)
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, test.statusCode, resp.StatusCode)
			if test.statusCode != http.StatusOK {
				return
			}
		})
	}
}

func TestPortalAPI_Router_getCollectionAPISpec_overrideServerAndAuth(t *testing.T) {
	spec, err := os.ReadFile("./testdata/openapi/spec.json")
	require.NoError(t, err)

	svcSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err = rw.Write(spec)
	}))

	tests := []struct {
		desc   string
		portal portal
		path   string
		want   string
	}{
		{
			desc: "without path prefix",
			portal: portal{
				APIPortal: hubv1alpha1.APIPortal{ObjectMeta: metav1.ObjectMeta{Name: "my-portal"}},
				Gateway: gateway{
					APIGateway: hubv1alpha1.APIGateway{
						ObjectMeta: metav1.ObjectMeta{Name: "my-gateway"},
						Status: hubv1alpha1.APIGatewayStatus{
							HubDomain: "majestic-beaver-123.hub-traefik.io",
						},
					},
					Collections: map[string]collection{
						"my-collection": {
							APIs: map[string]api{
								"my-api@my-ns": {
									API: hubv1alpha1.API{
										ObjectMeta: metav1.ObjectMeta{Name: "my-api", Namespace: "my-ns"},
										Spec: hubv1alpha1.APISpec{
											PathPrefix: "/api-prefix",
											Service: hubv1alpha1.APIService{
												Name:        "svc",
												Port:        hubv1alpha1.APIServiceBackendPort{Number: 80},
												OpenAPISpec: hubv1alpha1.OpenAPISpec{URL: svcSrv.URL},
											},
										},
									},
								},
							},
							authorizedGroups: []string{"supplier"},
						},
					},
				},
			},
			path: "/collections/my-collection/apis/my-api@my-ns",
			want: "./testdata/openapi/want-collection-no-path-prefix.json",
		},
		{
			desc: "with path prefix",
			portal: portal{
				APIPortal: hubv1alpha1.APIPortal{ObjectMeta: metav1.ObjectMeta{Name: "my-portal"}},
				Gateway: gateway{
					APIGateway: hubv1alpha1.APIGateway{
						ObjectMeta: metav1.ObjectMeta{Name: "my-gateway"},
						Status: hubv1alpha1.APIGatewayStatus{
							HubDomain: "majestic-beaver-123.hub-traefik.io",
						},
					},
					Collections: map[string]collection{
						"my-collection": {
							APICollection: hubv1alpha1.APICollection{
								ObjectMeta: metav1.ObjectMeta{Name: "my-collection"},
								Spec:       hubv1alpha1.APICollectionSpec{PathPrefix: "/collection-prefix"},
							},
							APIs: map[string]api{
								"my-api@my-ns": {
									API: hubv1alpha1.API{
										ObjectMeta: metav1.ObjectMeta{Name: "my-api", Namespace: "my-ns"},
										Spec: hubv1alpha1.APISpec{
											PathPrefix: "/api-prefix",
											Service: hubv1alpha1.APIService{
												Name:        "svc",
												Port:        hubv1alpha1.APIServiceBackendPort{Number: 80},
												OpenAPISpec: hubv1alpha1.OpenAPISpec{URL: svcSrv.URL},
											},
										},
									},
									authorizedGroups: []string{"supplier"},
								},
							},
							authorizedGroups: []string{"supplier"},
						},
					},
				},
			},
			path: "/collections/my-collection/apis/my-api@my-ns",
			want: "./testdata/openapi/want-collection-path-prefix.json",
		},
		{
			desc: "with custom domains",
			portal: portal{
				APIPortal: hubv1alpha1.APIPortal{ObjectMeta: metav1.ObjectMeta{Name: "my-portal"}},
				Gateway: gateway{
					APIGateway: hubv1alpha1.APIGateway{
						ObjectMeta: metav1.ObjectMeta{Name: "my-gateway"},
						Status: hubv1alpha1.APIGatewayStatus{
							HubDomain: "majestic-beaver-123.hub-traefik.io",
							CustomDomains: []string{
								"api.example.com",
								"www.api.example.com",
							},
						},
					},
					Collections: map[string]collection{
						"my-collection": {
							APICollection: hubv1alpha1.APICollection{
								ObjectMeta: metav1.ObjectMeta{Name: "my-collection"},
							},
							APIs: map[string]api{
								"my-api@my-ns": {
									API: hubv1alpha1.API{
										ObjectMeta: metav1.ObjectMeta{Name: "my-api", Namespace: "my-ns"},
										Spec: hubv1alpha1.APISpec{
											PathPrefix: "/api-prefix",
											Service: hubv1alpha1.APIService{
												Name:        "svc",
												Port:        hubv1alpha1.APIServiceBackendPort{Number: 80},
												OpenAPISpec: hubv1alpha1.OpenAPISpec{URL: svcSrv.URL},
											},
										},
									},
									authorizedGroups: []string{"supplier"},
								},
							},
							authorizedGroups: []string{"supplier"},
						},
					},
				},
			},
			path: "/collections/my-collection/apis/my-api@my-ns",
			want: "./testdata/openapi/want-collection-custom-domain.json",
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			a, err := NewPortalAPI(&test.portal, nil)
			require.NoError(t, err)
			a.httpClient = http.DefaultClient

			apiSrv := httptest.NewServer(a)

			req, err := http.NewRequest(http.MethodGet, apiSrv.URL+test.path, http.NoBody)
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, http.StatusOK, resp.StatusCode)

			got, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			wantSpec, err := os.ReadFile(test.want)
			require.NoError(t, err)

			assert.JSONEq(t, string(wantSpec), string(got))
		})
	}
}

func TestPortalAPI_Router_getAPISpec(t *testing.T) {
	tests := []struct {
		desc       string
		api        string
		wantURL    string
		statusCode int
	}{
		{
			desc:       "OpenAPI spec defined using an external URL",
			api:        "managers@people-ns",
			wantURL:    "http://my-oas-registry.example.com/artifacts/456",
			statusCode: http.StatusOK,
		},
		{
			desc:       "OpenAPI spec defined with a path on the service",
			api:        "notifications@default",
			wantURL:    "http://notifications-svc.default:8080/spec.json",
			statusCode: http.StatusOK,
		},
		{
			desc:       "OpenAPI spec defined with a path on the service and a specific port",
			api:        "metrics@default",
			wantURL:    "http://metrics-svc.default:9000/spec.json",
			statusCode: http.StatusOK,
		},
		{
			desc:       "No OpenAPI spec defined",
			api:        "health@default",
			wantURL:    "http://health-svc.default:8080/",
			statusCode: http.StatusOK,
		},
		{
			desc:       "Missing required group",
			api:        "api@default",
			wantURL:    "http://api-svc.default:8080/",
			statusCode: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			svcSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if r.URL.String() != test.wantURL {
					t.Logf("expected URL %q got %q", test.wantURL, r.URL.String())
					rw.WriteHeader(http.StatusNotFound)
					return
				}

				if err := json.NewEncoder(rw).Encode(openapi3.T{OpenAPI: "v3.0"}); err != nil {
					rw.WriteHeader(http.StatusInternalServerError)
				}
			}))
			a, err := NewPortalAPI(&testPortal, nil)
			require.NoError(t, err)
			a.httpClient = buildProxyClient(t, svcSrv.URL)

			apiSrv := httptest.NewServer(a)

			req, err := http.NewRequest(http.MethodGet, apiSrv.URL+"/apis/"+test.api, http.NoBody)
			require.NoError(t, err)

			req.Header.Add("Hub-Email", testEmail)
			req.Header.Add("Hub-Groups", "supplier")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, test.statusCode, resp.StatusCode)
			if test.statusCode != http.StatusOK {
				return
			}
		})
	}
}

func TestPortalAPI_Router_getAPISpec_overrideServerAndAuth(t *testing.T) {
	spec, err := os.ReadFile("./testdata/openapi/spec.json")
	require.NoError(t, err)

	wantSpec, err := os.ReadFile("./testdata/openapi/want-api.json")
	require.NoError(t, err)

	svcSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err = rw.Write(spec)
	}))

	p := portal{
		APIPortal: hubv1alpha1.APIPortal{ObjectMeta: metav1.ObjectMeta{Name: "my-portal"}},
		Gateway: gateway{
			APIGateway: hubv1alpha1.APIGateway{
				ObjectMeta: metav1.ObjectMeta{Name: "my-gateway"},
				Status:     hubv1alpha1.APIGatewayStatus{HubDomain: "majestic-beaver-123.hub-traefik.io"},
			},
			APIs: map[string]api{
				"my-api@my-ns": {
					API: hubv1alpha1.API{
						ObjectMeta: metav1.ObjectMeta{Name: "my-api", Namespace: "my-ns"},
						Spec: hubv1alpha1.APISpec{
							PathPrefix: "/api-prefix",
							Service: hubv1alpha1.APIService{
								Name:        "svc",
								Port:        hubv1alpha1.APIServiceBackendPort{Number: 80},
								OpenAPISpec: hubv1alpha1.OpenAPISpec{URL: svcSrv.URL},
							},
						},
					},
					authorizedGroups: []string{"supplier"},
				},
			},
		},
	}

	a, err := NewPortalAPI(&p, nil)
	require.NoError(t, err)
	a.httpClient = http.DefaultClient

	apiSrv := httptest.NewServer(a)

	uri := fmt.Sprintf("%s/apis/my-api@my-ns", apiSrv.URL)
	req, err := http.NewRequest(http.MethodGet, uri, http.NoBody)
	require.NoError(t, err)

	req.Header.Add("Hub-Email", testEmail)
	req.Header.Add("Hub-Groups", "supplier")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.JSONEq(t, string(wantSpec), string(got))
}

func buildProxyClient(t *testing.T, proxyURL string) *http.Client {
	t.Helper()

	u, err := url.Parse(proxyURL)
	require.NoError(t, err)

	return &http.Client{
		Transport: &http.Transport{
			Proxy: func(r *http.Request) (*url.URL, error) {
				r.URL.Host = u.Host

				return r.URL, nil
			},
		},
	}
}
