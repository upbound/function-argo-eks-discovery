// Package v1beta1 contains the input type for the Dummy Function
// +kubebuilder:object:generate=true
// +groupName=eksdiscovery.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// Response specifies Patch & Transform resource templates.
// +kubebuilder:resource:categories=crossplane
type Response struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Response must be a RunFunctionResponse in YAML/JSON form. The Function
	// will always return exactly this response when called.
	// +kubebuilder:pruning:PreserveUnknownFields
	Response runtime.RawExtension `json:"response"`
}
