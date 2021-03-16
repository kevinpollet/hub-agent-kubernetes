/*
 */

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/traefik/neo-agent/pkg/crd/api/neo/v1alpha1"
	scheme "github.com/traefik/neo-agent/pkg/crd/generated/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// IngressClassesGetter has a method to return a IngressClassInterface.
// A group's client should implement this interface.
type IngressClassesGetter interface {
	IngressClasses() IngressClassInterface
}

// IngressClassInterface has methods to work with IngressClass resources.
type IngressClassInterface interface {
	Create(ctx context.Context, ingressClass *v1alpha1.IngressClass, opts v1.CreateOptions) (*v1alpha1.IngressClass, error)
	Update(ctx context.Context, ingressClass *v1alpha1.IngressClass, opts v1.UpdateOptions) (*v1alpha1.IngressClass, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.IngressClass, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.IngressClassList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.IngressClass, err error)
	IngressClassExpansion
}

// ingressClasses implements IngressClassInterface
type ingressClasses struct {
	client rest.Interface
}

// newIngressClasses returns a IngressClasses
func newIngressClasses(c *NeoV1alpha1Client) *ingressClasses {
	return &ingressClasses{
		client: c.RESTClient(),
	}
}

// Get takes name of the ingressClass, and returns the corresponding ingressClass object, and an error if there is any.
func (c *ingressClasses) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.IngressClass, err error) {
	result = &v1alpha1.IngressClass{}
	err = c.client.Get().
		Resource("ingressclasses").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of IngressClasses that match those selectors.
func (c *ingressClasses) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.IngressClassList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.IngressClassList{}
	err = c.client.Get().
		Resource("ingressclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested ingressClasses.
func (c *ingressClasses) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("ingressclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a ingressClass and creates it.  Returns the server's representation of the ingressClass, and an error, if there is any.
func (c *ingressClasses) Create(ctx context.Context, ingressClass *v1alpha1.IngressClass, opts v1.CreateOptions) (result *v1alpha1.IngressClass, err error) {
	result = &v1alpha1.IngressClass{}
	err = c.client.Post().
		Resource("ingressclasses").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ingressClass).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a ingressClass and updates it. Returns the server's representation of the ingressClass, and an error, if there is any.
func (c *ingressClasses) Update(ctx context.Context, ingressClass *v1alpha1.IngressClass, opts v1.UpdateOptions) (result *v1alpha1.IngressClass, err error) {
	result = &v1alpha1.IngressClass{}
	err = c.client.Put().
		Resource("ingressclasses").
		Name(ingressClass.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(ingressClass).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the ingressClass and deletes it. Returns an error if one occurs.
func (c *ingressClasses) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("ingressclasses").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *ingressClasses) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("ingressclasses").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched ingressClass.
func (c *ingressClasses) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.IngressClass, err error) {
	result = &v1alpha1.IngressClass{}
	err = c.client.Patch(pt).
		Resource("ingressclasses").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
