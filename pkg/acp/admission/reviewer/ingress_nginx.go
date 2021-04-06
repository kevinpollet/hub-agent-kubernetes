package reviewer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/traefik/neo-agent/pkg/acp"
	"github.com/traefik/neo-agent/pkg/acp/admission/ingclass"
	admv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NginxIngress is a reviewer that handles Nginx Ingress resources.
type NginxIngress struct {
	agentAddress   string
	ingressClasses IngressClasses
	policies       PolicyGetter
}

// NewNginxIngress returns an Nginx ingress reviewer.
func NewNginxIngress(authServerAddr string, ingClasses IngressClasses, policies PolicyGetter) *NginxIngress {
	return &NginxIngress{
		agentAddress:   authServerAddr,
		ingressClasses: ingClasses,
		policies:       policies,
	}
}

// CanReview returns whether this reviewer can handle the given admission review request.
func (r NginxIngress) CanReview(ar admv1.AdmissionReview) (bool, error) {
	resource := ar.Request.Kind

	// Check resource type. Only continue if it's a legacy Ingress (<1.18) or an Ingress resource.
	if !isNetV1Ingress(resource) && !isNetV1Beta1Ingress(resource) && !isExtV1Beta1Ingress(resource) {
		return false, nil
	}

	ingClassName, ingClassAnno, err := parseIngressClass(ar.Request.Object.Raw)
	if err != nil {
		return false, fmt.Errorf("parse ingress class: %w", err)
	}

	defaultCtrlr, err := r.ingressClasses.GetDefaultController()
	if err != nil {
		return false, fmt.Errorf("get default controller: %w", err)
	}

	switch {
	case ingClassName != "":
		return isNginx(r.ingressClasses.GetController(ingClassName)), nil
	case ingClassAnno != "":
		if ingClassAnno == defaultAnnotationNginx {
			return true, nil
		}
		return isNginx(r.ingressClasses.GetController(ingClassAnno)), nil
	default:
		return isNginx(defaultCtrlr), nil
	}
}

// Review reviews the given admission review request and optionally returns the required patch.
func (r NginxIngress) Review(ctx context.Context, ar admv1.AdmissionReview) ([]byte, error) {
	l := log.Ctx(ctx).With().Str("reviewer", "NginxIngress").Logger()
	ctx = l.WithContext(ctx)

	log.Ctx(ctx).Info().Msg("Reviewing Ingress resource")

	// Fetch the metadata of the Ingress resource.
	var ing struct {
		Metadata metav1.ObjectMeta `json:"metadata"`
	}
	if err := json.Unmarshal(ar.Request.Object.Raw, &ing); err != nil {
		return nil, fmt.Errorf("unmarshal reviewed ingress metadata: %w", err)
	}

	polName := ing.Metadata.Annotations[AnnotationNeoAuth]

	var snippets nginxSnippets

	if polName == "" {
		log.Ctx(ctx).Debug().Msg("No ACP annotation found")
	} else {
		log.Ctx(ctx).Debug().Str("acp_name", polName).Msg("ACP annotation is present")

		canonicalPolName, err := acp.CanonicalName(polName, ing.Metadata.Namespace)
		if err != nil {
			return nil, err
		}

		polCfg, err := r.policies.GetConfig(canonicalPolName)
		if err != nil {
			return nil, err
		}

		snippets, err = genSnippets(canonicalPolName, polCfg, r.agentAddress)
		if err != nil {
			return nil, err
		}
	}
	snippets = mergeSnippets(snippets, ing.Metadata.Annotations)

	if noPatchRequired(ing.Metadata.Annotations, snippets) {
		log.Ctx(ctx).Debug().Str("acp_name", polName).Msg("No patch required")
		return nil, nil
	}

	setAnnotations(ing.Metadata.Annotations, snippets)

	log.Ctx(ctx).Info().Str("acp_name", polName).Msg("Patching resource")

	patch := []map[string]interface{}{
		{
			"op":    "replace",
			"path":  "/metadata/annotations",
			"value": ing.Metadata.Annotations,
		},
	}

	b, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("marshal ingress patch: %w", err)
	}
	return b, nil
}

func noPatchRequired(anno map[string]string, snippets nginxSnippets) bool {
	return anno["nginx.ingress.kubernetes.io/auth-url"] == snippets.AuthURL &&
		anno["nginx.ingress.kubernetes.io/configuration-snippet"] == snippets.ConfigurationSnippet &&
		anno["nginx.org/server-snippets"] == snippets.ServerSnippets &&
		anno["nginx.org/location-snippets"] == snippets.LocationSnippets
}

func setAnnotations(anno map[string]string, snippets nginxSnippets) {
	anno["nginx.ingress.kubernetes.io/auth-url"] = snippets.AuthURL
	anno["nginx.ingress.kubernetes.io/configuration-snippet"] = snippets.ConfigurationSnippet
	anno["nginx.org/server-snippets"] = snippets.ServerSnippets
	anno["nginx.org/location-snippets"] = snippets.LocationSnippets

	clearEmptyAnnotations(anno)
}

func clearEmptyAnnotations(anno map[string]string) {
	if anno["nginx.org/server-snippets"] == "" {
		delete(anno, "nginx.org/server-snippets")
	}
	if anno["nginx.org/location-snippets"] == "" {
		delete(anno, "nginx.org/location-snippets")
	}
	if anno["nginx.ingress.kubernetes.io/auth-url"] == "" {
		delete(anno, "nginx.ingress.kubernetes.io/auth-url")
	}
	if anno["nginx.ingress.kubernetes.io/configuration-snippet"] == "" {
		delete(anno, "nginx.ingress.kubernetes.io/configuration-snippet")
	}
}

func isNginx(ctrlr string) bool {
	return ctrlr == ingclass.ControllerTypeNginxOfficial || ctrlr == ingclass.ControllerTypeNginxCommunity
}