package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

const (
	// How the orchestrator waits without ever reading a Cloud Run *operation* — a
	// resource that can only be granted project-wide, which the deliberately
	// job-scoped orchestrator is not. It polls resources it already has access to
	// instead: the job (to confirm an update landed) and the blackboard (for the
	// result). pollInterval spaces the polls; the two timeouts bound them.
	pollInterval     = 3 * time.Second
	jobUpdateTimeout = 60 * time.Second
	runTimeout       = 20 * time.Minute
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

	if err := c.configureJob(ctx); err != nil {
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

	body, err := c.awaitResult(ctx, runID)
	if err != nil {
		return Result{}, err
	}
	var result Result
	if err := json.Unmarshal(body, &result); err != nil {
		return Result{}, fmt.Errorf("parse result of run %s: %w", runID, err)
	}
	return result, nil
}

// awaitResult polls the blackboard until the agent writes its result, or the deadline
// passes. The result the agent leaves *is* the run's completion signal — the source
// of truth anyway — and reading it needs nothing beyond the bucket access the
// orchestrator already holds, unlike waiting on the Cloud Run run operation.
func (c *CloudRun) awaitResult(ctx context.Context, runID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	for {
		body, found, err := c.store.GetResult(ctx, runID)
		if err != nil {
			return nil, fmt.Errorf("read result of run %s: %w", runID, err)
		}
		if found {
			return body, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("run %s produced no result within %s (see its Cloud Logging trace): %w", runID, runTimeout, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// imageRef names the prebuilt per-project image. CI keeps agent-<project>:latest
// current from the project's own Dockerfile; this only has to name it.
func (c *CloudRun) imageRef() string {
	return fmt.Sprintf("%s/%s/%s/agent-%s:latest", registryHost, c.gcpProject, registryRepo, c.project)
}

// execute starts one execution of the job. RUN_ID is the entire dispatch message —
// the claim check. It does not wait on the returned run operation (that read is
// project-wide only); the execution's outcome is observed on the blackboard instead,
// where the pod writes its result, and in Cloud Logging.
func (c *CloudRun) execute(ctx context.Context, runID string) error {
	_, err := c.jobs.RunJob(ctx, &runpb.RunJobRequest{
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
	return nil
}

// configureJob points the pre-existing agent job at this project's image and this
// run's env, and returns once the job reports that image.
//
// The job's *existence* belongs to Terraform (infra/agent.tf); the orchestrator only
// ever authors its contents. So this always updates and never creates — a job that
// is not there means infra was never applied, which is a real error to surface, not a
// cue to create one. The update is unconditional because the image is not the whole
// spec — the env, the secret and the service account live here too — and rewriting it
// every run is also what re-resolves the mutable :latest tag to the freshest build.
//
// It confirms the update by polling the job (a job-scoped read) rather than waiting
// on the returned operation: an operation can only be read with a project-wide grant,
// and the orchestrator is deliberately scoped to this one job.
func (c *CloudRun) configureJob(ctx context.Context) error {
	want := c.imageRef()
	job := c.jobSpec(want)
	job.Name = c.jobPath()

	if _, err := c.jobs.UpdateJob(ctx, &runpb.UpdateJobRequest{Job: job}); err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("agent job %q not found — has infra (agent.tf) been applied? %w", jobName, err)
		}
		return fmt.Errorf("update job: %w", err)
	}
	return c.awaitJobImage(ctx, want)
}

// awaitJobImage blocks until GetJob reports want, or the deadline passes. Reading the
// job is scoped to this one job; reading the update's operation would not be.
func (c *CloudRun) awaitJobImage(ctx context.Context, want string) error {
	ctx, cancel := context.WithTimeout(ctx, jobUpdateTimeout)
	defer cancel()
	for {
		job, err := c.jobs.GetJob(ctx, &runpb.GetJobRequest{Name: c.jobPath()})
		if err != nil {
			return fmt.Errorf("get job: %w", err)
		}
		if currentImage(job) == want {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("agent job did not report image %q within %s: %w", want, jobUpdateTimeout, ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// currentImage reads the image the job's task template currently declares, guarding
// the nested getters so an unexpectedly empty job reads as "" rather than panicking.
func currentImage(j *runpb.Job) string {
	containers := j.GetTemplate().GetTemplate().GetContainers()
	if len(containers) == 0 {
		return ""
	}
	return containers[0].GetImage()
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
