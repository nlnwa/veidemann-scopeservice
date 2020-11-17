package server

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/nlnwa/veidemann-api/go/commons/v1"
	"github.com/nlnwa/veidemann-api/go/config/v1"
	"github.com/nlnwa/veidemann-api/go/frontier/v1"
	"github.com/nlnwa/veidemann-api/go/scopechecker/v1"
	"reflect"
	"strings"
	"testing"
	"veidemann-scopeservice/pkg/script"
)

func init() {
	script.InitializeCanonicalizationProfiles(false)
}

func TestScopeCheckerServer_ScopeCheck(t *testing.T) {
	server := &ScopeCheckerService{}
	qUri := &frontier.QueuedUri{
		Id:                  "id1",
		ExecutionId:         "eid1",
		DiscoveredTimeStamp: ptypes.TimestampNow(),
		Sequence:            2,
		Uri:                 "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
		Surt:                "",
		Ip:                  "127.0.0.1",
		DiscoveryPath:       "RL",
		Referrer:            "http://foo.bar/",
		Cookies:             nil,
		Retries:             0,
		Annotation: []*config.Annotation{
			{Key: "testValue", Value: "True"},
		},
	}

	tests := []struct {
		name   string
		script string
		qUri   *frontier.QueuedUri
		debug  bool
		want   *scopechecker.ScopeCheckResponse
	}{
		{"1", "test(True).then(ChaffDetection)", qUri, false, &scopechecker.ScopeCheckResponse{
			Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
			ExcludeReason:   script.ChaffDetection.AsInt32(),
			IncludeCheckUri: "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
			Console:         "",
		}},
		{"2",
			"test(param(\"foo\"))", qUri, false,
			&scopechecker.ScopeCheckResponse{
				Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
				ExcludeReason:   script.RuntimeException.AsInt32(),
				IncludeCheckUri: "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
				Console:         "",
				Error: &commons.Error{
					Code:   -5,
					Msg:    "error executing scope script",
					Detail: "Traceback (most recent call last):\n  scope_script:1:11: in <toplevel>\n  <builtin>: in param\nError: no value with name 'foo'",
				},
			}},
		{"3",
			"test(", qUri, false,
			&scopechecker.ScopeCheckResponse{
				Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
				ExcludeReason:   script.RuntimeException.AsInt32(),
				IncludeCheckUri: "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
				Console:         "",
				Error: &commons.Error{
					Code:   -5,
					Msg:    "error parsing scope script",
					Detail: "scope_script:1:6: got end of file, want ')'",
				},
			}},
		{"4",
			"test(param(\"testValue\")).then(ChaffDetection).abort()", qUri, true,
			&scopechecker.ScopeCheckResponse{
				Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
				ExcludeReason:   script.ChaffDetection.AsInt32(),
				IncludeCheckUri: "http://foo.bar/aa%20bb/cc?foo&jsessionid=1",
				Console:         "scope_script:1:5 test(\"True\") match=True\nscope_script:1:30 match.then(ChaffDetection) status=ChaffDetection\n",
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &scopechecker.ScopeCheckRequest{
				QueuedUri:       tt.qUri,
				ScopeScriptName: "scope_script",
				ScopeScript:     tt.script,
				Debug:           tt.debug,
			}

			got, err := server.ScopeCheck(context.TODO(), request)
			if err != nil {
				t.Errorf("ScopeCheck() error = %v", err)
				return
			}
			if got.Evaluation != tt.want.Evaluation {
				t.Errorf("ScopeCheck() evaluation got = %v, want %v", got.Evaluation, tt.want.Evaluation)
			}
			if got.ExcludeReason != tt.want.ExcludeReason {
				t.Errorf("ScopeCheck() excludeReason got = %v, want %v", got.ExcludeReason, tt.want.ExcludeReason)
			}
			if got.IncludeCheckUri != tt.want.IncludeCheckUri {
				t.Errorf("ScopeCheck() includeCheckUri got = %v, want %v", got.IncludeCheckUri, tt.want.IncludeCheckUri)
			}
			if got.Console != tt.want.Console {
				t.Errorf("ScopeCheck() consoleLog \ngot:\n  %v\nwant:\n  %v",
					strings.ReplaceAll(got.Console, "\n", "\n  "),
					strings.ReplaceAll(tt.want.Console, "\n", "\n  "))
			}
			if !reflect.DeepEqual(got.Error, tt.want.Error) {
				t.Errorf("ScopeCheck() error \nGot:\n%v\nWant:\n%v\n", formatError(got.Error), formatError(tt.want.Error))
			}
		})
	}
}

func formatError(e *commons.Error) string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("    code: %v\n     msg: %v\n  detail: %v",
		e.Code, e.Msg, strings.ReplaceAll(e.Detail, "\n", "\n          "))
}
