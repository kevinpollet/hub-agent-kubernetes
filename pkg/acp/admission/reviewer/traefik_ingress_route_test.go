package reviewer_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traefik/neo-agent/pkg/acp"
	"github.com/traefik/neo-agent/pkg/acp/admission"
	"github.com/traefik/neo-agent/pkg/acp/admission/reviewer"
	"github.com/traefik/neo-agent/pkg/acp/basicauth"
	"github.com/traefik/neo-agent/pkg/acp/digestauth"
	"github.com/traefik/neo-agent/pkg/acp/jwt"
	traefikv1alpha1 "github.com/traefik/neo-agent/pkg/crd/api/traefik/v1alpha1"
	traefikkubemock "github.com/traefik/neo-agent/pkg/crd/generated/client/traefik/clientset/versioned/fake"
	admv1 "k8s.io/api/admission/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestTraefikIngressRoute_HandleACPName(t *testing.T) {
	factory := func(policies reviewer.PolicyGetter) admission.Reviewer {
		fwdAuthMdlwrs := reviewer.NewFwdAuthMiddlewares("", policies, traefikkubemock.NewSimpleClientset().TraefikV1alpha1())
		return reviewer.NewTraefikIngressRoute(fwdAuthMdlwrs)
	}

	ingressHandleACPName(t, factory)
}

func TestTraefikIngressRoute_CanReviewChecksKind(t *testing.T) {
	tests := []struct {
		desc      string
		kind      metav1.GroupVersionKind
		canReview bool
	}{
		{
			desc: "can review traefik.containo.us v1alpha1 IngressRoute",
			kind: metav1.GroupVersionKind{
				Group:   "traefik.containo.us",
				Version: "v1alpha1",
				Kind:    "IngressRoute",
			},
			canReview: true,
		},
		{
			desc: "can't review invalid traefik.containo.us IngressRoute version",
			kind: metav1.GroupVersionKind{
				Group:   "traefik.containo.us",
				Version: "invalid",
				Kind:    "IngressRoute",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid traefik.containo.us IngressRoute Ingress group",
			kind: metav1.GroupVersionKind{
				Group:   "invalid",
				Version: "v1alpha1",
				Kind:    "IngressRoute",
			},
			canReview: false,
		},
		{
			desc: "can't review non traefik.containo.us IngressRoute resources",
			kind: metav1.GroupVersionKind{
				Group:   "networking.k8s.io",
				Version: "v1",
				Kind:    "NetworkPolicy",
			},
			canReview: false,
		},
		{
			desc: "can review extensions v1beta1 Ingresses",
			kind: metav1.GroupVersionKind{
				Group:   "extensions",
				Version: "v1beta1",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid extensions Ingress version",
			kind: metav1.GroupVersionKind{
				Group:   "extensions",
				Version: "invalid",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid v1beta1 Ingress group",
			kind: metav1.GroupVersionKind{
				Group:   "invalid",
				Version: "v1beta1",
				Kind:    "Ingress",
			},
			canReview: false,
		},
		{
			desc: "can't review invalid extension v1beta1 resource",
			kind: metav1.GroupVersionKind{
				Group:   "extensions",
				Version: "v1beta1",
				Kind:    "Invalid",
			},
			canReview: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			policies := func(canonicalName string) *acp.Config {
				return nil
			}
			fwdAuthMdlwrs := reviewer.NewFwdAuthMiddlewares("", policyGetterMock(policies), nil)
			review := reviewer.NewTraefikIngressRoute(fwdAuthMdlwrs)

			var ing netv1.Ingress
			b, err := json.Marshal(ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Kind: test.kind,
					Object: runtime.RawExtension{
						Raw: b,
					},
				},
			}

			ok, err := review.CanReview(ar)
			require.NoError(t, err)
			assert.Equal(t, test.canReview, ok)
		})
	}
}

func TestTraefikIngressRoute_ReviewAddsAuthentication(t *testing.T) {
	tests := []struct {
		desc                    string
		config                  *acp.Config
		oldIng                  traefikv1alpha1.IngressRoute
		ing                     traefikv1alpha1.IngressRoute
		wantPatch               []traefikv1alpha1.Route
		wantAuthResponseHeaders []string
	}{
		{
			desc: "add JWT authentication",
			config: &acp.Config{JWT: &jwt.Config{
				ForwardHeaders: map[string]string{
					"fwdHeader": "claim",
				},
			}},
			oldIng: traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-old-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{
						{
							Middlewares: []traefikv1alpha1.MiddlewareRef{
								{
									Name:      "custom-middleware",
									Namespace: "test",
								},
								{
									Name:      "zz-my-old-policy-test",
									Namespace: "test",
								},
							},
						},
					},
				},
			},
			ing: traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{
						{
							Match:    "match",
							Kind:     "kind",
							Priority: 2,
							Services: []traefikv1alpha1.Service{
								{
									LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{Name: "Name", Namespace: "ns", Kind: "kind"},
								},
							},
							Middlewares: []traefikv1alpha1.MiddlewareRef{
								{
									Name:      "custom-middleware",
									Namespace: "test",
								},
								{
									Name:      "zz-my-old-policy-test",
									Namespace: "test",
								},
							},
						},
					},
				},
			},
			wantPatch: []traefikv1alpha1.Route{
				{
					Match:    "match",
					Kind:     "kind",
					Priority: 2,
					Services: []traefikv1alpha1.Service{
						{
							LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{Name: "Name", Namespace: "ns", Kind: "kind"},
						},
					},
					Middlewares: []traefikv1alpha1.MiddlewareRef{
						{
							Name:      "custom-middleware",
							Namespace: "test",
						},
						{
							Name:      "zz-my-policy-test",
							Namespace: "test",
						},
					},
				},
			},
			wantAuthResponseHeaders: []string{"fwdHeader"},
		},
		{
			desc: "add Basic authentication",
			config: &acp.Config{BasicAuth: &basicauth.Config{
				StripAuthorizationHeader: true,
				ForwardUsernameHeader:    "User",
			}},
			oldIng: traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-old-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{{}},
				},
			},
			ing: traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{{}},
				},
			},
			wantPatch: []traefikv1alpha1.Route{
				{
					Middlewares: []traefikv1alpha1.MiddlewareRef{
						{
							Name:      "zz-my-policy-test",
							Namespace: "test",
						},
					},
				},
			},
			wantAuthResponseHeaders: []string{"User", "Authorization"},
		},
		{
			desc: "add Digest authentication",
			config: &acp.Config{DigestAuth: &digestauth.Config{
				StripAuthorizationHeader: true,
				ForwardUsernameHeader:    "User",
			}},
			oldIng: traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-old-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{{}},
				},
			},
			ing: traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{{}},
				},
			},
			wantPatch: []traefikv1alpha1.Route{
				{
					Middlewares: []traefikv1alpha1.MiddlewareRef{
						{
							Name:      "zz-my-policy-test",
							Namespace: "test",
						},
					},
				},
			},
			wantAuthResponseHeaders: []string{"User", "Authorization"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			traefikClientSet := traefikkubemock.NewSimpleClientset()
			policies := func(canonicalName string) *acp.Config {
				return test.config
			}
			fwdAuthMdlwrs := reviewer.NewFwdAuthMiddlewares("", policyGetterMock(policies), traefikClientSet.TraefikV1alpha1())
			rev := reviewer.NewTraefikIngressRoute(fwdAuthMdlwrs)

			oldB, err := json.Marshal(test.oldIng)
			require.NoError(t, err)

			b, err := json.Marshal(test.ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: b,
					},
					OldObject: runtime.RawExtension{
						Raw: oldB,
					},
				},
			}

			p, err := rev.Review(context.Background(), ar)
			assert.NoError(t, err)
			assert.NotNil(t, p)

			var patches []map[string]interface{}
			err = json.Unmarshal(p, &patches)
			require.NoError(t, err)

			assert.Equal(t, 1, len(patches))
			assert.Equal(t, "replace", patches[0]["op"])
			assert.Equal(t, "/spec/routes", patches[0]["path"])

			b, err = json.Marshal(patches[0]["value"])
			require.NoError(t, err)

			var middlewares []traefikv1alpha1.Route
			err = json.Unmarshal(b, &middlewares)
			require.NoError(t, err)

			for i, route := range middlewares {
				if !reflect.DeepEqual(route, test.wantPatch[i]) {
					t.Fail()
				}
			}

			m, err := traefikClientSet.TraefikV1alpha1().Middlewares("test").Get(context.Background(), "zz-my-policy-test", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, m)

			assert.Equal(t, test.wantAuthResponseHeaders, m.Spec.ForwardAuth.AuthResponseHeaders)
		})
	}
}

func TestTraefikIngressRoute_ReviewUpdatesExistingMiddleware(t *testing.T) {
	tests := []struct {
		desc                    string
		config                  *acp.Config
		wantAuthResponseHeaders []string
	}{
		{
			desc: "Update middleware with JWT configuration",
			config: &acp.Config{
				JWT: &jwt.Config{
					StripAuthorizationHeader: true,
				},
			},
			wantAuthResponseHeaders: []string{"Authorization"},
		},
		{
			desc: "Update middleware with basic configuration",
			config: &acp.Config{
				BasicAuth: &basicauth.Config{
					StripAuthorizationHeader: true,
				},
			},
			wantAuthResponseHeaders: []string{"Authorization"},
		},
		{
			desc: "Update middleware with digest configuration",
			config: &acp.Config{
				DigestAuth: &digestauth.Config{
					StripAuthorizationHeader: true,
				},
			},
			wantAuthResponseHeaders: []string{"Authorization"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			middleware := traefikv1alpha1.Middleware{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "zz-my-policy-test",
					Namespace: "test",
				},
				Spec: traefikv1alpha1.MiddlewareSpec{
					ForwardAuth: &traefikv1alpha1.ForwardAuth{
						AuthResponseHeaders: []string{"fwdHeader"},
					},
				},
			}
			traefikClientSet := traefikkubemock.NewSimpleClientset(&middleware)
			policies := func(canonicalName string) *acp.Config {
				return test.config
			}
			fwdAuthMdlwrs := reviewer.NewFwdAuthMiddlewares("", policyGetterMock(policies), traefikClientSet.TraefikV1alpha1())
			rev := reviewer.NewTraefikIngressRoute(fwdAuthMdlwrs)

			ing := traefikv1alpha1.IngressRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "test",
					Annotations: map[string]string{
						reviewer.AnnotationNeoAuth: "my-policy@test",
						"custom-annotation":        "foobar",
					},
				},
				Spec: traefikv1alpha1.IngressRouteSpec{
					Routes: []traefikv1alpha1.Route{
						{
							Match:    "match",
							Kind:     "kind",
							Priority: 2,
							Services: []traefikv1alpha1.Service{
								{
									LoadBalancerSpec: traefikv1alpha1.LoadBalancerSpec{Name: "Name", Namespace: "ns", Kind: "kind"},
								},
							},
							Middlewares: nil,
						},
					},
				},
			}
			b, err := json.Marshal(ing)
			require.NoError(t, err)

			ar := admv1.AdmissionReview{
				Request: &admv1.AdmissionRequest{
					Object: runtime.RawExtension{
						Raw: b,
					},
				},
			}

			m, err := traefikClientSet.TraefikV1alpha1().Middlewares("test").Get(context.Background(), "zz-my-policy-test", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, m)
			assert.Equal(t, []string{"fwdHeader"}, m.Spec.ForwardAuth.AuthResponseHeaders)

			p, err := rev.Review(context.Background(), ar)
			assert.NoError(t, err)
			assert.NotNil(t, p)

			m, err = traefikClientSet.TraefikV1alpha1().Middlewares("test").Get(context.Background(), "zz-my-policy-test", metav1.GetOptions{})
			assert.NoError(t, err)
			assert.NotNil(t, m)

			assert.Equal(t, test.wantAuthResponseHeaders, m.Spec.ForwardAuth.AuthResponseHeaders)
		})
	}
}