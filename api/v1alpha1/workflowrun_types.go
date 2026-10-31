package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phases of a WorkflowRun
const (
	PhasePending   = "Pending"
	PhaseRunning   = "Running"
	PhaseSucceeded = "Succeeded"
	PhaseFailed    = "Failed"
)

// WorkflowRunSpec defines the desired state of WorkflowRun
type WorkflowRunSpec struct {
	// WorkflowTemplateRef references the WorkflowTemplate to execute
	WorkflowTemplateRef string `json:"workflowTemplateRef"`

	// Parameters are key-value pairs passed to the workflow execution
	// These typically include context like git repo, branch, commit, environment, etc.
	// They are exposed to each step as environment variables.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// WorkflowRunStatus defines the observed state of WorkflowRun
type WorkflowRunStatus struct {
	// Phase is the overall state of the run
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed
	Phase string `json:"phase,omitempty"`

	// CurrentStep is the index of the step being executed
	CurrentStep int32 `json:"currentStep,omitempty"`

	// Message is a human-readable description of the current state
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// WorkflowRun is the Schema for the workflowruns API
type WorkflowRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowRunSpec   `json:"spec,omitempty"`
	Status WorkflowRunStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkflowRunList contains a list of WorkflowRun
type WorkflowRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkflowRun{}, &WorkflowRunList{})
}
