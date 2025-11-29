/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	"github.com/jnnkrdb/echosec/internal/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterObjectStatus defines the observed state of ClusterObject.
type ClusterObjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ClusterObject resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	LatestErrors []ReconcileError `json:"latestErrors"`

	// +default:numerical=0
	ReconcileIndex int `json:"reconcileIndex"`
}

// this is a status object, which is used to list the errors made during the last 10 runs
type ReconcileError struct {
	DateTime time.Time `json:"dateTime"`
	//+optional
	Namespace string `json:"namespace,omitempty"`
	Error     string `json:"error"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// ClusterObject is the Schema for the clusterobjects API
type ClusterObject struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// +optional
	RegexRules NamespaceRegexRules `json:"namespaceRegexRules"`

	// +required
	// +kubebuilder:pruning:PreserveUnknownFields
	Resource unstructured.Unstructured `json:"resource"`

	// status defines the observed state of ClusterObject
	// +optional
	Status ClusterObjectStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ClusterObjectList contains a list of ClusterObject
type ClusterObjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterObject `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterObject{}, &ClusterObjectList{})
}

// -------------------------------------------------------- declaring funcs

type ReconcilerClient struct{}

// remove errors, older than a constant amount of time (15min)
func (co *ClusterObject) RemoveOldErrors(ctx context.Context) {

	const timeDuration time.Duration = time.Duration(15 * time.Minute)

	var _log = log.FromContext(ctx)
	c, ok := ctx.Value(ReconcilerClient{}).(client.Client)
	if !ok {
		_log.Error(fmt.Errorf("couldn't get reconciler from context"), "receive client error", `ctx.Value(ReconcilerClient{})`, ctx.Value(ReconcilerClient{}))
		return
	}

	for i, e := range co.Status.LatestErrors {
		if timeDuration < time.Since(co.Status.LatestErrors[i].DateTime) {
			co.Status.LatestErrors = pkg.RemoveFromSlice(co.Status.LatestErrors, e)
		}
	}
	if err := c.Status().Update(ctx, co, &client.SubResourceUpdateOptions{}); err != nil {
		_log.Error(err, "unable to remove old errors from status subresource")
	}
}

// add an error to the status subresource of the object
func (co *ClusterObject) AddErrorToStatus(ctx context.Context, namespace string, err error) {
	var _log = log.FromContext(ctx)
	c, ok := ctx.Value(ReconcilerClient{}).(client.Client)
	if !ok {
		_log.Error(fmt.Errorf("couldn't get reconciler from context"), "receive client error", `ctx.Value(ReconcilerClient{})`, ctx.Value(ReconcilerClient{}))
		return
	}

	co.Status.LatestErrors = append(co.Status.LatestErrors, ReconcileError{
		Namespace: namespace,
		DateTime:  time.Now(),
		Error:     err.Error(),
	})

	if err := c.Status().Update(ctx, co, &client.SubResourceUpdateOptions{}); err != nil {
		_log.Error(err, "unable to apply error message to status subresource")
	}
}
