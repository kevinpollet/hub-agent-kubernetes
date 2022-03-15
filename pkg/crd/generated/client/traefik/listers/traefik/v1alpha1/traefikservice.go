/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/traefik/hub-agent-kubernetes/pkg/crd/api/traefik/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// TraefikServiceLister helps list TraefikServices.
// All objects returned here must be treated as read-only.
type TraefikServiceLister interface {
	// List lists all TraefikServices in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.TraefikService, err error)
	// TraefikServices returns an object that can list and get TraefikServices.
	TraefikServices(namespace string) TraefikServiceNamespaceLister
	TraefikServiceListerExpansion
}

// traefikServiceLister implements the TraefikServiceLister interface.
type traefikServiceLister struct {
	indexer cache.Indexer
}

// NewTraefikServiceLister returns a new TraefikServiceLister.
func NewTraefikServiceLister(indexer cache.Indexer) TraefikServiceLister {
	return &traefikServiceLister{indexer: indexer}
}

// List lists all TraefikServices in the indexer.
func (s *traefikServiceLister) List(selector labels.Selector) (ret []*v1alpha1.TraefikService, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.TraefikService))
	})
	return ret, err
}

// TraefikServices returns an object that can list and get TraefikServices.
func (s *traefikServiceLister) TraefikServices(namespace string) TraefikServiceNamespaceLister {
	return traefikServiceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// TraefikServiceNamespaceLister helps list and get TraefikServices.
// All objects returned here must be treated as read-only.
type TraefikServiceNamespaceLister interface {
	// List lists all TraefikServices in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.TraefikService, err error)
	// Get retrieves the TraefikService from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.TraefikService, error)
	TraefikServiceNamespaceListerExpansion
}

// traefikServiceNamespaceLister implements the TraefikServiceNamespaceLister
// interface.
type traefikServiceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all TraefikServices in the indexer for a given namespace.
func (s traefikServiceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.TraefikService, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.TraefikService))
	})
	return ret, err
}

// Get retrieves the TraefikService from the indexer for a given namespace and name.
func (s traefikServiceNamespaceLister) Get(name string) (*v1alpha1.TraefikService, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("traefikservice"), name)
	}
	return obj.(*v1alpha1.TraefikService), nil
}
