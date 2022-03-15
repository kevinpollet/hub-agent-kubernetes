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

// MiddlewareLister helps list Middlewares.
// All objects returned here must be treated as read-only.
type MiddlewareLister interface {
	// List lists all Middlewares in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Middleware, err error)
	// Middlewares returns an object that can list and get Middlewares.
	Middlewares(namespace string) MiddlewareNamespaceLister
	MiddlewareListerExpansion
}

// middlewareLister implements the MiddlewareLister interface.
type middlewareLister struct {
	indexer cache.Indexer
}

// NewMiddlewareLister returns a new MiddlewareLister.
func NewMiddlewareLister(indexer cache.Indexer) MiddlewareLister {
	return &middlewareLister{indexer: indexer}
}

// List lists all Middlewares in the indexer.
func (s *middlewareLister) List(selector labels.Selector) (ret []*v1alpha1.Middleware, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Middleware))
	})
	return ret, err
}

// Middlewares returns an object that can list and get Middlewares.
func (s *middlewareLister) Middlewares(namespace string) MiddlewareNamespaceLister {
	return middlewareNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MiddlewareNamespaceLister helps list and get Middlewares.
// All objects returned here must be treated as read-only.
type MiddlewareNamespaceLister interface {
	// List lists all Middlewares in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Middleware, err error)
	// Get retrieves the Middleware from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Middleware, error)
	MiddlewareNamespaceListerExpansion
}

// middlewareNamespaceLister implements the MiddlewareNamespaceLister
// interface.
type middlewareNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Middlewares in the indexer for a given namespace.
func (s middlewareNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Middleware, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Middleware))
	})
	return ret, err
}

// Get retrieves the Middleware from the indexer for a given namespace and name.
func (s middlewareNamespaceLister) Get(name string) (*v1alpha1.Middleware, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("middleware"), name)
	}
	return obj.(*v1alpha1.Middleware), nil
}
