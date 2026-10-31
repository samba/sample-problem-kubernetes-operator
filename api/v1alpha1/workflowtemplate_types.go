package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTemplateSpec defines the desired state of WorkflowTemplate
type WorkflowTemplateSpec struct {
	// Steps defines the workflow as an ordered list of steps.
	// Steps run one at a time, in list order.
	// +kubebuilder:validation:MinItems=1
	Steps []Step `json:"steps"`
}

// Step is a single unit of work, run as one Kubernetes Job
type Step struct {
	// Name of the step
	Name string `json:"name"`

	// Image is the container image the step runs
	Image string `json:"image"`

	// Command overrides the image entrypoint
	// +optional
	Command []string `json:"command,omitempty"`

	// Args are passed to the command
	// +optional
	Args []string `json:"args,omitempty"`

	// TimeoutSeconds bounds how long the step may run
	// +optional
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`
}

// WorkflowTemplateStatus defines the observed state of WorkflowTemplate
type WorkflowTemplateStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WorkflowTemplate is the Schema for the workflowtemplates API
type WorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowTemplateSpec   `json:"spec,omitempty"`
	Status WorkflowTemplateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkflowTemplateList contains a list of WorkflowTemplate
type WorkflowTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkflowTemplate{}, &WorkflowTemplateList{})
}
