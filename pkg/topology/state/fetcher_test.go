package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	hubkubemock "github.com/traefik/hub-agent-kubernetes/pkg/crd/generated/client/hub/clientset/versioned/fake"
	traefikkubemock "github.com/traefik/hub-agent-kubernetes/pkg/crd/generated/client/traefik/clientset/versioned/fake"
	netv1 "k8s.io/api/networking/v1"
	netv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubemock "k8s.io/client-go/kubernetes/fake"
)

func Test_watchAll_handlesUnsupportedVersions(t *testing.T) {
	tests := []struct {
		desc          string
		serverVersion string
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			desc:    "Empty",
			wantErr: assert.Error,
		},
		{
			desc:          "Malformed version",
			serverVersion: "foobar",
			wantErr:       assert.Error,
		},
		{
			desc:          "Unsupported version",
			serverVersion: "v1.13",
			wantErr:       assert.Error,
		},
		{
			desc:          "Supported version",
			serverVersion: "v1.16",
			wantErr:       assert.NoError,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			kubeClient := kubemock.NewSimpleClientset()
			hubClient := hubkubemock.NewSimpleClientset()
			traefikClient := traefikkubemock.NewSimpleClientset()

			_, err := watchAll(context.Background(), kubeClient, hubClient, traefikClient, test.serverVersion, "cluster-id")

			test.wantErr(t, err)
		})
	}
}

func Test_watchAll_handlesAllIngressAPIVersions(t *testing.T) {
	tests := []struct {
		desc          string
		serverVersion string
		want          map[string]*Ingress
	}{
		{
			desc:          "v1.16",
			serverVersion: "v1.16",
			want: map[string]*Ingress{
				"myIngress_netv1beta1@myns.ingress.networking.k8s.io": {
					ResourceMeta: ResourceMeta{
						Kind:      "Ingress",
						Group:     "networking.k8s.io",
						Name:      "myIngress_netv1beta1",
						Namespace: "myns",
					},
					IngressMeta: IngressMeta{
						ClusterID: "cluster-id",
					},
				},
			},
		},
		{
			desc:          "v1.18",
			serverVersion: "v1.18",
			want: map[string]*Ingress{
				"myIngress_netv1beta1@myns.ingress.networking.k8s.io": {
					ResourceMeta: ResourceMeta{
						Kind:      "Ingress",
						Group:     "networking.k8s.io",
						Name:      "myIngress_netv1beta1",
						Namespace: "myns",
					},
					IngressMeta: IngressMeta{
						ClusterID: "cluster-id",
					},
				},
			},
		},
		{
			desc:          "v1.18.10",
			serverVersion: "v1.18.10",
			want: map[string]*Ingress{
				"myIngress_netv1beta1@myns.ingress.networking.k8s.io": {
					ResourceMeta: ResourceMeta{
						Kind:      "Ingress",
						Group:     "networking.k8s.io",
						Name:      "myIngress_netv1beta1",
						Namespace: "myns",
					},
					IngressMeta: IngressMeta{
						ClusterID: "cluster-id",
					},
				},
			},
		},
		{
			desc:          "v1.19",
			serverVersion: "v1.19",
			want: map[string]*Ingress{
				"myIngress_netv1@myns.ingress.networking.k8s.io": {
					ResourceMeta: ResourceMeta{
						Kind:      "Ingress",
						Group:     "networking.k8s.io",
						Name:      "myIngress_netv1",
						Namespace: "myns",
					},
					IngressMeta: IngressMeta{
						ClusterID: "cluster-id",
					},
				},
			},
		},
		{
			desc:          "v1.22",
			serverVersion: "v1.22",
			want: map[string]*Ingress{
				"myIngress_netv1@myns.ingress.networking.k8s.io": {
					ResourceMeta: ResourceMeta{
						Kind:      "Ingress",
						Group:     "networking.k8s.io",
						Name:      "myIngress_netv1",
						Namespace: "myns",
					},
					IngressMeta: IngressMeta{
						ClusterID: "cluster-id",
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			k8sObjects := []runtime.Object{
				&netv1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "myns",
						Name:      "myIngress_netv1beta1",
					},
				},
				&netv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "myns",
						Name:      "myIngress_netv1",
					},
				},
			}

			kubeClient := kubemock.NewSimpleClientset(k8sObjects...)
			hubClient := hubkubemock.NewSimpleClientset()
			traefikClient := traefikkubemock.NewSimpleClientset()

			f, err := watchAll(context.Background(), kubeClient, hubClient, traefikClient, test.serverVersion, "cluster-id")
			require.NoError(t, err)

			got, err := f.getIngresses("cluster-id")
			require.NoError(t, err)

			assert.Equal(t, test.want, got)
		})
	}
}

func Test_getOverview(t *testing.T) {
	state := Cluster{
		Ingresses: map[string]*Ingress{
			"name@namespace.kind.group": {},
		},
		IngressRoutes: map[string]*IngressRoute{
			"name@namespace.kind.group": {},
		},
		Services: map[string]*Service{
			"name@namespace": {},
		},
		IngressControllers: map[string]*IngressController{
			"name@namespace":  {Type: IngressControllerTypeTraefik},
			"name2@namespace": {Type: IngressControllerTypeTraefik},
			"name3@namespace": {Type: IngressControllerTypeHAProxyCommunity},
		},
	}

	overview := getOverview(&state)

	want := Overview{
		IngressCount:           2,
		ServiceCount:           1,
		IngressControllerTypes: []string{IngressControllerTypeHAProxyCommunity, IngressControllerTypeTraefik},
	}

	assert.Equal(t, want, overview)
}
