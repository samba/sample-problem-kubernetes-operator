# Workflow Operator - Design & Implementation Challenge

## Overview

This repository contains a **limited first implementation** of a Kubernetes operator that manages workflow executions, similar to CI/CD pipelines (like GitLab CI). It works for the simple case, and it does not yet do what the product needs.

> **This is a sample problem for a hiring interview.** The code is intentionally incomplete and is **not production ready** in its current state. A candidate (you) is expected to complete it.

**Your job is to add the features below and make the result production ready.** That means:

- **Take part in requirements gathering and refinement.** The requirements are deliberately high-level. Ask questions, state your assumptions, and settle ambiguities before you build.
- **Implement the features.**
- **Resolve any deficiencies in the code.** This includes code you did not write. You own the result for the whole repository.
- **Add tests.**

---

## What exists today

Two CRDs in `workflow.example.com/v1alpha1`.

**WorkflowTemplate** is an ordered list of steps. Each step runs one container image:
```yaml
apiVersion: workflow.example.com/v1alpha1
kind: WorkflowTemplate
metadata:
  name: my-workflow
spec:
  steps:
    - name: compile
      image: golang:1.21
      command: ["go", "build", "./..."]
      timeoutSeconds: 600
    - name: test
      image: golang:1.21
      command: ["go", "test", "./..."]
```

**WorkflowRun** is one execution of a template:
```yaml
apiVersion: workflow.example.com/v1alpha1
kind: WorkflowRun
metadata:
  name: my-workflow-run
spec:
  workflowTemplateRef: my-workflow
  parameters:
    GIT_REPO: "https://github.com/myorg/myapp"
    ENVIRONMENT: "staging"
```

The controller runs the steps one at a time, in order, as Kubernetes Jobs. Parameters are passed to each step as environment variables. The run's `status` shows a phase and the index of the current step.

## Target behavior

This is what the operator must do once your work is complete.

1. **Stages run sequentially.** A workflow is a list of stages. Stage N+1 starts only after every job in stage N has succeeded.
2. **Jobs within a stage run in parallel.** All jobs of a stage are created together.
3. **All jobs must succeed.** If any job fails, the stage and the workflow fail.
4. **Two kinds of job:**
   - **Direct JobSpec:** a native Kubernetes `JobSpec`, used as given. Users need full control of the pod (volumes, service account, resources, security context and so on).
   - **Script-based:** the user names a script in a ConfigMap. The controller changes the JobSpec to mount the ConfigMap and run the script.

### Observability
- Users can see workflow progress.
- Users can tell which jobs succeeded and which failed.
- Users can see timing information.

---

## 🚀 START HERE: Design Questions (Answer First!)

**Before writing any code**, answer these questions. Your answers will:
1. Guide your implementation decisions
2. Be discussed in your interview
3. Provide context for AI agents assisting you

Brief answers are fine. Choose and explain.

---

### Question 1: Reading the existing code

Read `api/` and `controllers/` first. Then answer:

```
What in the existing design makes the target behavior hard to add?
(API shape, status, controller flow. Name the specific things.)


What are you assuming about anything that already exists around this operator?
(Objects, users, tooling.) How does that affect your approach?


```

---

### Question 2: Job identification and status

The target behavior has several jobs running at once in one stage. You need to find "all jobs of WorkflowRun X, stage 2" and show per-job results to the user.

```
How do you name and label Jobs?


What does WorkflowRun status look like? (paste the Go struct or describe it)


What happens to the Jobs when the WorkflowRun is deleted?
[ ] Deleted (owner references)  [ ] Kept (forensics)  [ ] Other:
Why:
```

---

### Question 3: Parallel stages

```
One job in a stage fails while others are still running. What do you do with the others?


How do you prevent a stage from being skipped or started twice
if reconcile runs several times, or runs with stale data?


Should a failed workflow be retryable?  [ ] Yes [ ] No   How?
```

---

### Question 4: Script jobs

Define the requirements yourself before you implement.

```
Steps (brief):
1.
2.
3.

If the JobSpec has 2 containers, mount the script in:
[ ] All containers  [ ] First only  [ ] Make it an error

Which shell runs the script, and why?

What happens when the ConfigMap or the key is missing? When do you find out?

What could go wrong (security, correctness) with scripts coming from a ConfigMap?
```

---

### Question 5: Failure handling and operation

```
A) Top 2 errors to handle well (pick from: Template not found, ConfigMap missing,
   Job creation fails, Job fails with error code):
1.
2.
Why these?

B) The operator runs for months on a shared cluster. What would you want to know
   or limit before you trust it? (Name 2-3 things.)


C) For debugging "workflow stuck", what status info is most critical?
1.
2.
3.
```

---

## 🤖 AI Prompt Templates (Copy-Paste These!)

**After answering the questions above, use these prompts with your AI agent:**

### Step 1: Review the existing implementation

```
I'm extending a Kubernetes WorkflowRun operator. Read api/ and controllers/.

Target behavior: [paste the Target behavior section]

1. Summarize what the code does today, step by step.
2. List what must change to reach the target behavior.
```

---

### Step 2: Design and implement the extension

```
Using this analysis: [paste the answers from Step 1]

And these design decisions:
- Q1 Constraints and assumptions: [paste your answer]
- Q2 Job identification and status: [paste your answer]
- Q3 Parallel stages: [paste your answer]
- Q4 Script jobs: [paste your answer]

Implement the extension. Show me the API changes before the controller changes.
Use controller-runtime best practices.
```

---

### Step 3: Generate Tests

```
Given my implementation, generate tests for:

1. A basic workflow: one stage, jobs succeed, workflow completes
2. Multi-stage: three stages, verify progression between stages
3. Job failure: a job fails in stage 2, verify the workflow fails

Use controller-runtime's fake client.
Include setup, execution, and assertions.
```

---

### Step 4: Fix Issues

```
[If you hit errors, paste the error and ask:]

I'm getting this error: [paste error]

In this code: [paste relevant code]

Based on my design decision [reference Q#], how should I fix this?
```

---

## Getting Started

### Prerequisites
- Go 1.27+
- Kubernetes cluster (kind, minikube, or similar for local testing)
- kubectl configured

### Build and Run Locally

```bash
# Install CRDs
make install

# Run the operator locally
make run

# In another terminal, create sample resources
kubectl apply -f config/samples/
```

### Build Docker Image

```bash
make docker-build
```

## Your Mission

### 1. Design (via the questions above)

Document:
- What choices you made and why
- What trade-offs you considered
- What assumptions you're making
- What edge cases you're aware of
- What you'd improve given more time

### 2. Implement

- Extend the API and controller to the target behavior
- Implement stages and parallel jobs
- Implement script jobs
- Handle errors appropriately
- Keep the operator working for what already works

### 3. Add Tests

**This published draft of the challenge intentionally includes no tests.** Expect to write your own, with the aid of AI, to validate the scope of the intended functionality.

Create `controllers/workflowrun_controller_test.go` with at least the 3 scenarios in Step 3 (basic, multi-stage, job failure).  
Add tests for the other behavior you changed.

## Success Criteria

Your solution will be evaluated on:

1. **Design Quality** (35%) - via the questions in this README
   - ✅ Thoughtful answers to all 5 questions
   - ✅ Brief but clear reasoning
   - ✅ Understanding of trade-offs

2. **AI Collaboration & Scoping** (30%)
   - ✅ Scoped the work sensibly and said what you left out
   - ✅ Effective use of the provided AI prompts
   - ✅ Can explain AI-generated code, and challenge it

3. **Implementation** (25%)
   - ✅ Core workflow works (70-80% is fine!)
   - ✅ Design reflected in code
   - ✅ Compiles and runs

4. **Testing** (10%)
   - ✅ At least 3 tests written: basic workflow, multi-stage progression, job failure
   - ✅ Tests pass

**Target: a 70-80% working solution**  
**Working + explainable > perfect**

## Tips

- Use `kubectl get workflowruns -w` to watch workflow execution
- Check job status with `kubectl get jobs`
- Use `kubectl describe workflowrun <name>` to see status details
- Controller logs will show reconciliation activity
- The controller-runtime documentation is your friend

## Deliverables

Submit a Git repository with:

1. **✅ Completed Design Questions in this README** (REQUIRED)
2. **💻 Working Implementation**
   - Stages with parallel jobs
   - Script job support (basic version OK)
   - Parameter passing
3. **✅ Tests you wrote** (at least the 3 scenarios)
4. **📝 Brief Summary**
   - What works, what doesn't
   - Key assumptions

## Important Notes

- **This is not a "fill in the blanks" exercise** - you need to make design decisions
- **There is no single "right" answer** - we want to understand your thinking
- **Document your assumptions** - if requirements are ambiguous, state your assumptions
- **Quality over quantity** - a well-designed, partially complete solution is better than a buggy full implementation
- **Ask questions if needed** - in a real scenario, you'd clarify requirements with stakeholders

Good luck! We're excited to see your solution and discuss your approach.
