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
	v1alpha1 "github.com/traefik/hub-agent-kubernetes/pkg/crd/api/hub/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AccessControlPolicyLister helps list AccessControlPolicies.
// All objects returned here must be treated as read-only.
type AccessControlPolicyLister interface {
	// List lists all AccessControlPolicies in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AccessControlPolicy, err error)
	// AccessControlPolicies returns an object that can list and get AccessControlPolicies.
	AccessControlPolicies(namespace string) AccessControlPolicyNamespaceLister
	AccessControlPolicyListerExpansion
}

// accessControlPolicyLister implements the AccessControlPolicyLister interface.
type accessControlPolicyLister struct {
	indexer cache.Indexer
}

// NewAccessControlPolicyLister returns a new AccessControlPolicyLister.
func NewAccessControlPolicyLister(indexer cache.Indexer) AccessControlPolicyLister {
	return &accessControlPolicyLister{indexer: indexer}
}

// List lists all AccessControlPolicies in the indexer.
func (s *accessControlPolicyLister) List(selector labels.Selector) (ret []*v1alpha1.AccessControlPolicy, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AccessControlPolicy))
	})
	return ret, err
}

// AccessControlPolicies returns an object that can list and get AccessControlPolicies.
func (s *accessControlPolicyLister) AccessControlPolicies(namespace string) AccessControlPolicyNamespaceLister {
	return accessControlPolicyNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// AccessControlPolicyNamespaceLister helps list and get AccessControlPolicies.
// All objects returned here must be treated as read-only.
type AccessControlPolicyNamespaceLister interface {
	// List lists all AccessControlPolicies in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.AccessControlPolicy, err error)
	// Get retrieves the AccessControlPolicy from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.AccessControlPolicy, error)
	AccessControlPolicyNamespaceListerExpansion
}

// accessControlPolicyNamespaceLister implements the AccessControlPolicyNamespaceLister
// interface.
type accessControlPolicyNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all AccessControlPolicies in the indexer for a given namespace.
func (s accessControlPolicyNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.AccessControlPolicy, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.AccessControlPolicy))
	})
	return ret, err
}

// Get retrieves the AccessControlPolicy from the indexer for a given namespace and name.
func (s accessControlPolicyNamespaceLister) Get(name string) (*v1alpha1.AccessControlPolicy, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("accesscontrolpolicy"), name)
	}
	return obj.(*v1alpha1.AccessControlPolicy), nil
}
