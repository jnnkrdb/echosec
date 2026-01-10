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

	// +optional
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" protobuf:"bytes,4,opt,name=labelSelector"`

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

// -------------------------------------------------------- helpers
type ClusterObjectContextKey struct{}

func (co *ClusterObject) IntoContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ClusterObjectContextKey{}, co)
}

func (co *ClusterObject) FromContext(ctx context.Context) error {
	if co, ok := ctx.Value(ClusterObjectContextKey{}).(*ClusterObject); !ok {
		return fmt.Errorf("invalid value from context: %v", co)
	}
	return nil
}

const (
	Condition_Ready = "Ready"
)

func (co *ClusterObject) FindCondition(ctx context.Context, conditionType string) *metav1.Condition {

	var _log = log.FromContext(ctx).WithValues("conditionType", conditionType)

	_log.V(5).Info("requested condition")

	for i := range co.Status.Conditions {

		if co.Status.Conditions[i].Type == conditionType {

			_log.V(5).Info("condition found", "condition", co.Status.Conditions[i])

			return &co.Status.Conditions[i]
		}
	}

	_log.V(5).Info("condition not found")

	return nil
}

func (co *ClusterObject) SetCondition(ctx context.Context, c client.Client,
	t string, s metav1.ConditionStatus, r string, mf string, a ...any) error {

	var _log = log.FromContext(ctx)

	var _condition = co.FindCondition(ctx, t)

	// if there is no condition with the specified type, then create a new condition and
	// add it to the list of conditions
	if _condition == nil {

		c := metav1.Condition{
			Type:               t,
			Status:             s,
			ObservedGeneration: co.GetGeneration(),
			LastTransitionTime: metav1.Now(),
			Reason:             r,
			Message:            fmt.Sprintf(mf, a...),
		}

		_log.V(5).Info("adding condition", "condition", c)

		co.Status.Conditions = append(co.Status.Conditions, c)

	} else {

		// cnage values of the given condition
		_condition.LastTransitionTime = func() metav1.Time {
			if _condition.Status != s {

				return metav1.Now()
			}
			return _condition.LastTransitionTime
		}()
		_condition.Status = s
		_condition.ObservedGeneration = co.GetGeneration()
		_condition.Reason = r
		_condition.Message = fmt.Sprintf(mf, a...)

		_log.V(5).Info("updated condition", "condition", *_condition)

	}

	return c.Status().Update(ctx, co, &client.SubResourceUpdateOptions{})
}
