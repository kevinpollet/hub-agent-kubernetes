package state

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	acpfake "github.com/traefik/neo-agent/pkg/crd/generated/client/clientset/versioned/fake"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubemock "k8s.io/client-go/kubernetes/fake"
)

func Test_WatchAllHandlesUnsupportedVersions(t *testing.T) {
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
			acpClient := acpfake.NewSimpleClientset()

			_, err := watchAll(context.Background(), kubeClient, acpClient, test.serverVersion)

			test.wantErr(t, err)
		})
	}
}

func Test_WatchAllHandlesAllIngressAPIVersions(t *testing.T) {
	tests := []struct {
		desc          string
		serverVersion string
		want          map[string]*Ingress
	}{
		{
			desc:          "v1.16",
			serverVersion: "v1.16",
			want: map[string]*Ingress{
				"myIngress_netv1beta1@myns": {
					Namespace: "myns",
					Name:      "myIngress_netv1beta1",
					ClusterID: "myClusterID",
				},
			},
		},
		{
			desc:          "v1.18",
			serverVersion: "v1.18",
			want: map[string]*Ingress{
				"myIngress_netv1beta1@myns": {
					Name:      "myIngress_netv1beta1",
					Namespace: "myns",
					ClusterID: "myClusterID",
				},
			},
		},
		{
			desc:          "v1.19",
			serverVersion: "v1.19",
			want: map[string]*Ingress{
				"myIngress_netv1@myns": {
					Name:      "myIngress_netv1",
					Namespace: "myns",
					ClusterID: "myClusterID",
				},
			},
		},
		{
			desc:          "v1.22",
			serverVersion: "v1.22",
			want: map[string]*Ingress{
				"myIngress_netv1@myns": {
					Name:      "myIngress_netv1",
					Namespace: "myns",
					ClusterID: "myClusterID",
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			k8sObjects := []runtime.Object{
				&v1beta1.Ingress{
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
			acpClient := acpfake.NewSimpleClientset()

			f, err := watchAll(context.Background(), kubeClient, acpClient, test.serverVersion)
			require.NoError(t, err)

			got, err := f.getIngresses("myClusterID")
			require.NoError(t, err)

			assert.Equal(t, test.want, got)
		})
	}
}