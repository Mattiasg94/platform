package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	run "cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	jobName        = "agent"
	jobRegion      = "us-central1"
	agentServiceAc = "agent-job@%s.iam.gserviceaccount.com"
	claudeToken    = "claude-code-oauth-token" // infra/secrets.tf

	registryHost = "us-central1-docker.pkg.dev"
	registryRepo = "platform" // infra/registry.tf
)

var _ Runner = (*CloudRun)(nil)

// CloudRun executes the agent as a Cloud Run Job against a *prebuilt* per-project
// image. It builds nothing: the image agent-<project> is produced ahead of time
// by CI (.github/workflows/agent-images.yml) from the project's own Dockerfile
// layered with the agent harness (ADR-0009), and this only references it by name.
// The pod still learns only its run id and still talks through the blackboard, and
// still runs *as* the agent-job service account — it gets its bucket access and
// its token from the platform, not from us.
type CloudRun struct {
	jobs       *run.JobsClient
	store      store
	workspace  string // the cloned project tree shipped to the agent
	project    string // names the prebuilt image: agent-<project>
	bucket     string
	gcpProject string
}

func NewCloudRun(ctx context.Context, workspace, project, bucket, gcpProject string, blackboard store) (*CloudRun, error) {
	jobs, err := run.NewJobsClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cloud run client: %w", err)
	}
	return &CloudRun{
		jobs:       jobs,
		store:      blackboard,
		workspace:  workspace,
		project:    project,
		bucket:     bucket,
		gcpProject: gcpProject,
	}, nil
}

// Run points the agent job at the project's prebuilt image, hands the pod its
// task and workspace through the blackboard, and blocks until it finishes.
func (c *CloudRun) Run(ctx context.Context, prompt string) (Result, error) {
	runID := newRunID()
	log.Printf("run %s", runID)

	if err := c.reconcileJob(ctx); err != nil {
		return Result{}, err
	}
	if err := c.store.PutTask(ctx, runID, prompt); err != nil {
		return Result{}, err
	}
	if err := c.store.PutWorkspace(ctx, runID, c.workspace); err != nil {
		return Result{}, err
	}

	if err := c.execute(ctx, runID); err != nil {
		return Result{}, err
	}

	body, err := c.store.GetResult(ctx, runID)
	if err != nil {
		return Result{}, fmt.Errorf("run %s produced no result (see its Cloud Logging trace): %w", runID, err)
	}
	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		return Result{}, fmt.Errorf("parse result of run %s: %w", runID, err)
	}
	return result, nil
}

// imageRef names the prebuilt per-project image. CI keeps agent-<project>:latest
// current from the project's own Dockerfile; this only has to name it.
func (c *CloudRun) imageRef() string {
	return fmt.Sprintf("%s/%s/%s/agent-%s:latest", registryHost, c.gcpProject, registryRepo, c.project)
}

// execute starts one execution of the job and blocks until it finishes. RUN_ID is
// the entire dispatch message — the claim check.
func (c *CloudRun) execute(ctx context.Context, runID string) error {
	op, err := c.jobs.RunJob(ctx, &runpb.RunJobRequest{
		Name: c.jobPath(),
		Overrides: &runpb.RunJobRequest_Overrides{
			ContainerOverrides: []*runpb.RunJobRequest_Overrides_ContainerOverride{{
				Env: []*runpb.EnvVar{
					{Name: "RUN_ID", Values: &runpb.EnvVar_Value{Value: runID}},
				},
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("start execution: %w", err)
	}

	execution, err := op.Wait(ctx)
	if err != nil {
		// A failed execution is not an error here: the pod may still have written a
		// result explaining itself, and that is more informative than an exit code.
		log.Printf("execution finished unhappily: %v", err)
		return nil
	}
	log.Printf("execution %s: %d succeeded, %d failed",
		execution.GetName(), execution.GetSucceededCount(), execution.GetFailedCount())
	return nil
}

// reconcileJob makes the job definition match what this run needs, creating it the
// first time and writing it every time after.
//
// The write is unconditional on purpose. Skipping it looks like a free
// optimisation, but the image is not the whole spec — the env, the secrets and the
// service account live here too, and a change to any of them would then never
// reach the job. An UpdateJob is idempotent and costs a couple of seconds; a job
// definition that silently lags the code costs an afternoon. (Now that the image
// is a stable prebuilt ref rather than a content hash only this process knows, the
// definition could move to Terraform — a later simplification.)
func (c *CloudRun) reconcileJob(ctx context.Context) error {
	job := c.jobSpec(c.imageRef())

	_, err := c.jobs.GetJob(ctx, &runpb.GetJobRequest{Name: c.jobPath()})
	switch {
	case status.Code(err) == codes.NotFound:
		op, err := c.jobs.CreateJob(ctx, &runpb.CreateJobRequest{
			Parent: c.parent(),
			JobId:  jobName,
			Job:    job,
		})
		if err != nil {
			return fmt.Errorf("create job: %w", err)
		}
		_, err = op.Wait(ctx)
		return err
	case err != nil:
		return fmt.Errorf("get job: %w", err)
	}

	job.Name = c.jobPath()
	op, err := c.jobs.UpdateJob(ctx, &runpb.UpdateJobRequest{Job: job})
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	_, err = op.Wait(ctx)
	return err
}

func (c *CloudRun) jobSpec(image string) *runpb.Job {
	return &runpb.Job{
		Template: &runpb.ExecutionTemplate{
			Template: &runpb.TaskTemplate{
				// The agent is not idempotent — it calls a model and opens work. A
				// retried task would burn tokens redoing something that may have half
				// happened. Failure is the orchestrator's to see, not Cloud Run's to
				// paper over.
				Retries: &runpb.TaskTemplate_MaxRetries{MaxRetries: 0},
				Containers: []*runpb.Container{{
					Image: image,
					Env: []*runpb.EnvVar{
						{Name: "RUNS_BUCKET", Values: &runpb.EnvVar_Value{Value: c.bucket}},
						// No credentials and no project id: the pod runs as agent-job, and
						// the metadata server answers both. This is what the local Docker
						// runner had to fake with a mounted file.
						{Name: "HOME", Values: &runpb.EnvVar_Value{Value: "/tmp"}},
						// The harness bills a subscription, not the API. This is the whole
						// switch — and it only works because ANTHROPIC_API_KEY is absent:
						// the CLI ranks the key above the token, so if both were listed here
						// the key would win silently and we would still be paying. Cloud Run
						// passes only what this slice names, so leaving it out is enough.
						{
							Name: "CLAUDE_CODE_OAUTH_TOKEN",
							Values: &runpb.EnvVar_ValueSource{ValueSource: &runpb.EnvVarSource{
								SecretKeyRef: &runpb.SecretKeySelector{
									Secret:  claudeToken,
									Version: "latest",
								},
							}},
						},
					},
				}},
				ServiceAccount: fmt.Sprintf(agentServiceAc, c.gcpProject),
			},
		},
	}
}

func (c *CloudRun) parent() string {
	return fmt.Sprintf("projects/%s/locations/%s", c.gcpProject, jobRegion)
}

func (c *CloudRun) jobPath() string {
	return fmt.Sprintf("%s/jobs/%s", c.parent(), jobName)
}
