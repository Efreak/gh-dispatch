package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/cli/cli/v2/pkg/cmd/run/shared"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/spf13/cobra"
)

type workflowDispatchRequest struct {
	Inputs interface{} `json:"inputs"`
}

// workflowCmd represents the workflow subcommand
var workflowCmd = &cobra.Command{
	Use:     "workflow",
	Short:   `The 'workflow' subcommand triggers workflow dispatch events`,
	Long:    `The 'workflow' subcommand triggers workflow dispatch events`,
	Example: `TODO`,
}

func workflowDispatchRun(opts *dispatchOptions) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(workflowDispatchRequest{
		Inputs: opts.Inputs,
	})
	if err != nil {
		return err
	}

	client, err := gh.RESTClient(&api.ClientOptions{
		Transport: opts.HTTPTransport,
		AuthToken: opts.AuthToken,
	})
	if err != nil {
		return err
	}

	var in interface{}
	err = client.Post(fmt.Sprintf("repos/%s/actions/workflows/%s/dispatches", opts.Repo, opts.WorkflowID), &buf, &in)
	if err != nil {
		return err
	}

	runID, err := getWorkflowDispatchRunID(client, opts.Repo, opts.WorkflowID)
	if err != nil {
		return err
	}

	run, err := getRun(client, opts.Repo, runID)
	if err != nil {
		return fmt.Errorf("failed to get run: %w", err)
	}

	cs := opts.IO.ColorScheme()
	annotationCache := map[int64][]shared.Annotation{}
	out := &bytes.Buffer{}
	opts.IO.StartAlternateScreenBuffer()

	for run.Status != shared.Completed {
		// Write to a temporary buffer to reduce total number of fetches
		run, err = renderRun(out, opts.IO, client, opts.Repo, run, annotationCache)
		if err != nil {
			return err
		}

		if run.Status == shared.Completed {
			break
		}

		// If not completed, refresh the screen buffer and write the temporary buffer to stdout
		opts.IO.RefreshScreen()

		interval := 3
		fmt.Fprintln(opts.IO.Out, cs.Boldf("Refreshing run status every %d seconds. Press Ctrl+C to quit.", interval))
		fmt.Fprintln(opts.IO.Out)
		fmt.Fprintln(opts.IO.Out, cs.Boldf("https://github.com/%s/actions/runs/%d", opts.Repo, runID))
		fmt.Fprintln(opts.IO.Out)

		_, err = io.Copy(opts.IO.Out, out)
		out.Reset()
		if err != nil {
			break
		}

		duration, err := time.ParseDuration(fmt.Sprintf("%ds", interval))
		if err != nil {
			return fmt.Errorf("could not parse interval: %w", err)
		}
		time.Sleep(duration)
	}

	opts.IO.StopAlternateScreenBuffer()

	return nil
}

func getWorkflowDispatchRunID(client api.RESTClient, repo, workflow string) (int64, error) {
	for {
		var wRuns workflowRunsResponse
		err := client.Get(fmt.Sprintf("repos/%s/actions/runs?name=%s&event=workflow_dispatch", repo, workflow), &wRuns)
		if err != nil {
			return 0, err
		}

		if wRuns.WorkflowRuns[0].Status != shared.Completed {
			return wRuns.WorkflowRuns[0].ID, nil
		}
	}
}

func init() {
	rootCmd.AddCommand(workflowCmd)
}
