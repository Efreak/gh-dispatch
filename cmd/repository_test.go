package cmd

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/stretchr/testify/assert"
)

func TestRepositoryDispatchRun(t *testing.T) {
	repo := "OWNER/REPO"
	tests := []struct {
		name      string
		opts      *repositoryDispatchOptions
		httpStubs func(*httpmock.Registry)
		wantErr   bool
		errMsg    string
		wantOut   string
	}{
		{
			name: "no specified workflow",
			opts: &repositoryDispatchOptions{
				eventType: "hello",
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("POST", fmt.Sprintf("repos/%s/dispatches", repo)),
					httpmock.StringResponse("{}"))
			},
			wantOut: "",
		}, {
			name: "specified workflow",
			opts: &repositoryDispatchOptions{
				eventType: "hello",
				workflow:  "foo",
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("POST", fmt.Sprintf("repos/%s/dispatches", repo)),
					httpmock.StringResponse("{}"))

				v := url.Values{}
				v.Set("event", "repository_dispatch")

				reg.Register(
					httpmock.QueryMatcher("GET", fmt.Sprintf("repos/%s/actions/runs", repo), v),
					httpmock.StringResponse(getWorkflowRunsResponse))

				reg.Register(
					httpmock.REST("GET", fmt.Sprintf("repos/%s/actions/runs/123", repo)),
					httpmock.StringResponse(`{
						"id": 123
					}`))

				reg.Register(
					httpmock.REST("GET", fmt.Sprintf("repos/%s/actions/runs/123", repo)),
					httpmock.StringResponse(`{
						"id": 123
					}`))

				reg.Register(
					httpmock.REST("GET", fmt.Sprintf("repos/%s/actions/runs/123", repo)),
					httpmock.StringResponse(`{
						"id": 123,
						"status": "completed"
					}`))

				reg.Register(
					httpmock.REST("GET", fmt.Sprintf("repos/%s/actions/runs/123/attempts/1/jobs", repo)),
					httpmock.StringResponse(getJobsResponse))

				reg.Register(
					httpmock.REST("GET", fmt.Sprintf("repos/%s/actions/runs/123/attempts/1/jobs", repo)),
					httpmock.StringResponse(getJobsResponse))

				reg.Register(
					httpmock.REST("GET", fmt.Sprintf("repos/%s/check-runs/123/annotations", repo)),
					httpmock.StringResponse("[]"))
			},
			wantOut: "Refreshing run status every 3 seconds. Press Ctrl+C to quit.\n\nhttps://github.com/OWNER/REPO/actions/runs/123\n\n\nJOBS\n✓ build in 1m59s (ID 123)\n  ✓ Run actions/checkout@v2\n  ✓ Test\n",
		}, {
			name: "malformed JSON response",
			opts: &repositoryDispatchOptions{
				eventType: "hello",
			},
			httpStubs: func(reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("POST", fmt.Sprintf("repos/%s/dispatches", repo)),
					httpmock.StringResponse("{"))
			},
			wantOut: "",
			wantErr: true,
			errMsg:  "unexpected end of JSON input",
		}}

	for _, tt := range tests {
		reg := &httpmock.Registry{}
		tt.httpStubs(reg)

		ios, _, stdout, _ := iostreams.Test()
		ios.SetStdoutTTY(false)
		ios.SetAlternateScreenBufferEnabled(false)

		tt.opts.repo = repo
		tt.opts.io = ios
		tt.opts.httpTransport = reg
		tt.opts.authToken = "123"

		t.Run(tt.name, func(t *testing.T) {
			err := repositoryDispatchRun(tt.opts)

			if tt.wantErr {
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			if got := stdout.String(); got != tt.wantOut {
				t.Errorf("got stdout:\n%q\nwant:\n%q", got, tt.wantOut)
			}
			reg.Verify(t)
		})
	}
}
