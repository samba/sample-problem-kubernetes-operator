package controllers

import (
	"context"
	"fmt"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	workflowv1alpha1 "github.com/samba/sample-problem-kubernetes-operator/api/v1alpha1"
)

const (
	labelWorkflowRun = "workflowrun"
	labelStep        = "step"
)

// stepResult is the outcome of checking the Jobs of a step
type stepResult int

const (
	stepRunning stepResult = iota
	stepSucceeded
	stepFailed
)

// WorkflowRunReconciler reconciles a WorkflowRun object
type WorkflowRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=workflow.example.com,resources=workflowruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workflow.example.com,resources=workflowruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workflow.example.com,resources=workflowruns/finalizers,verbs=update
//+kubebuilder:rbac:groups=workflow.example.com,resources=workflowtemplates,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile runs the steps of a WorkflowRun one at a time, in order.
func (r *WorkflowRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the WorkflowRun instance
	workflowRun := &workflowv1alpha1.WorkflowRun{}
	err := r.Get(ctx, req.NamespacedName, workflowRun)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Nothing more to do for a finished run
	if workflowRun.Status.Phase == workflowv1alpha1.PhaseSucceeded ||
		workflowRun.Status.Phase == workflowv1alpha1.PhaseFailed {
		return ctrl.Result{}, nil
	}

	// Fetch the WorkflowTemplate
	template := &workflowv1alpha1.WorkflowTemplate{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      workflowRun.Spec.WorkflowTemplateRef,
		Namespace: workflowRun.Namespace,
	}, template)
	if err != nil {
		log.Error(err, "Failed to get WorkflowTemplate")
		return ctrl.Result{}, err
	}

	// Initialize status on first reconcile
	if workflowRun.Status.Phase == "" {
		workflowRun.Status.Phase = workflowv1alpha1.PhaseRunning
		workflowRun.Status.CurrentStep = 0
		workflowRun.Status.Message = "workflow started"
		if err := r.Status().Update(ctx, workflowRun); err != nil {
			return ctrl.Result{}, err
		}
	}

	stepIndex := int(workflowRun.Status.CurrentStep)
	step := template.Spec.Steps[stepIndex]

	if _, err := r.createJobFromTemplate(ctx, workflowRun, &step, stepIndex); err != nil {
		log.Error(err, "Failed to create Job", "step", step.Name)
		return ctrl.Result{}, err
	}

	result, failedJob, err := r.checkStepStatus(ctx, workflowRun, stepIndex)
	if err != nil {
		return ctrl.Result{}, err
	}

	switch result {
	case stepFailed:
		workflowRun.Status.Phase = workflowv1alpha1.PhaseFailed
		workflowRun.Status.Message = fmt.Sprintf("step %q failed (job %s)", step.Name, failedJob)
	case stepSucceeded:
		if stepIndex+1 >= len(template.Spec.Steps) {
			workflowRun.Status.Phase = workflowv1alpha1.PhaseSucceeded
			workflowRun.Status.Message = "all steps succeeded"
		} else {
			workflowRun.Status.CurrentStep++
			workflowRun.Status.Message = fmt.Sprintf("step %q succeeded", step.Name)
		}
	default:
		return ctrl.Result{}, nil
	}

	log.Info("Updating WorkflowRun status", "phase", workflowRun.Status.Phase, "step", workflowRun.Status.CurrentStep)
	return ctrl.Result{}, r.Status().Update(ctx, workflowRun)
}

// checkStepStatus inspects the Jobs created for a step and reports whether the step
// is still running, has succeeded, or has failed. When a Job failed, its name is returned.
func (r *WorkflowRunReconciler) checkStepStatus(ctx context.Context, workflowRun *workflowv1alpha1.WorkflowRun, stepIndex int) (stepResult, string, error) {
	jobs := &batchv1.JobList{}
	if err := r.List(ctx, jobs,
		client.InNamespace(workflowRun.Namespace),
		client.MatchingLabels{
			labelWorkflowRun: workflowRun.Name,
			labelStep:        fmt.Sprintf("%d", stepIndex),
		}); err != nil {
		return stepRunning, "", err
	}

	for _, job := range jobs.Items {
		if job.Status.Failed > 0 {
			return stepFailed, job.Name, nil
		}
		if job.Status.Succeeded == 0 {
			return stepRunning, "", nil
		}
	}
	return stepSucceeded, "", nil
}

// createJobFromTemplate creates the Job for a step, or returns it if it already exists
func (r *WorkflowRunReconciler) createJobFromTemplate(ctx context.Context, workflowRun *workflowv1alpha1.WorkflowRun, step *workflowv1alpha1.Step, stepIndex int) (*batchv1.Job, error) {
	jobName := fmt.Sprintf("%s-%d-%s", workflowRun.Name, stepIndex, step.Name)

	// Check if job already exists
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: workflowRun.Namespace}, existingJob)
	if err == nil {
		return existingJob, nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	// Pass workflow parameters to the step as environment variables
	keys := make([]string, 0, len(workflowRun.Spec.Parameters))
	for k := range workflowRun.Spec.Parameters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	env := make([]corev1.EnvVar, 0, len(keys))
	for _, k := range keys {
		env = append(env, corev1.EnvVar{Name: k, Value: workflowRun.Spec.Parameters[k]})
	}

	jobSpec := batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				Containers: []corev1.Container{{
					Name:    "step",
					Image:   step.Image,
					Command: step.Command,
					Args:    step.Args,
					Env:     env,
				}},
			},
		},
	}
	if step.TimeoutSeconds > 0 {
		deadline := int64(step.TimeoutSeconds) * int64(time.Second)
		jobSpec.ActiveDeadlineSeconds = &deadline
	}

	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: workflowRun.Namespace,
			Labels: map[string]string{
				labelWorkflowRun: workflowRun.Name,
				labelStep:        fmt.Sprintf("%d", stepIndex),
			},
			// Jobs are deleted along with the WorkflowRun
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: workflowv1alpha1.GroupVersion.String(),
				Kind:       "WorkflowRun",
				Name:       workflowRun.Name,
				UID:        workflowRun.UID,
			}},
		},
		Spec: jobSpec,
	}

	if err := r.Create(ctx, k8sJob); err != nil {
		if errors.IsAlreadyExists(err) {
			return k8sJob, nil
		}
		return nil, err
	}

	return k8sJob, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflowv1alpha1.WorkflowRun{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
