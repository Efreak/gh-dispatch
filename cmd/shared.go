package cmd

import (
	"fmt"
	"io"
	"net/http"

	"github.com/cli/cli/v2/pkg/cmd/run/shared"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/go-gh/pkg/api"
)

type workflowRun struct {
	ID         int64         `json:"id"`
	WorkflowID int           `json:"workflow_id"`
	Name       string        `json:"name"`
	Status     shared.Status `json:"status"`
	Conclusion string        `json:"conclusion"`
}

type workflowRunsResponse struct {
	WorkflowRuns []workflowRun `json:"workflow_runs"`
}

type dispatchOptions struct {
	Repo          string
	Inputs        interface{}
	ClientPayload interface{}
	EventType     string
	WorkflowID    string
	Workflow      string
	IO            *iostreams.IOStreams
	HTTPTransport http.RoundTripper
	AuthToken     string
}

func renderRun(out io.Writer, io *iostreams.IOStreams, client api.RESTClient, repo string, run *shared.Run, annotationCache map[int64][]shared.Annotation) (*shared.Run, error) {
	cs := io.ColorScheme()

	run, err := getRun(client, repo, run.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	jobs, err := getJobs(client, repo, run.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	var annotations []shared.Annotation
	var annotationErr error
	var as []shared.Annotation
	for _, job := range jobs {
		if as, ok := annotationCache[job.ID]; ok {
			annotations = as
			continue
		}

		as, annotationErr = getAnnotations(client, repo, job)
		if annotationErr != nil {
			break
		}
		annotations = append(annotations, as...)

		if job.Status != shared.InProgress {
			annotationCache[job.ID] = annotations
		}
	}

	if annotationErr != nil {
		return nil, fmt.Errorf("failed to get annotations: %w", annotationErr)
	}

	fmt.Fprintln(out)

	if len(jobs) == 0 {
		return run, nil
	}

	fmt.Fprintln(out, cs.Bold("JOBS"))
	fmt.Fprintln(out, shared.RenderJobs(cs, jobs, true))

	if len(annotations) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, cs.Bold("ANNOTATIONS"))
		fmt.Fprintln(out, shared.RenderAnnotations(cs, annotations))
	}

	return run, nil
}

func getRun(client api.RESTClient, repo string, runID int64) (*shared.Run, error) {
	var result shared.Run
	err := client.Get(fmt.Sprintf("repos/%s/actions/runs/%d", repo, runID), &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func getJobs(client api.RESTClient, repo string, runID int64) ([]shared.Job, error) {
	var result shared.JobsPayload
	err := client.Get(fmt.Sprintf("repos/%s/actions/runs/%d/attempts/1/jobs", repo, runID), &result)
	if err != nil {
		return nil, err
	}

	return result.Jobs, nil
}

func getAnnotations(client api.RESTClient, repo string, job shared.Job) ([]shared.Annotation, error) {
	var result []*shared.Annotation
	err := client.Get(fmt.Sprintf("repos/%s/check-runs/%d/annotations", repo, job.ID), &result)
	if err != nil {
		return nil, err
	}

	annotations := []shared.Annotation{}
	for _, annotation := range result {
		annotation.JobName = job.Name
		annotations = append(annotations, *annotation)
	}

	return annotations, nil
}
