package server

import (
	"context"
	"github.com/nlnwa/veidemann-api/go/config/v1"
	"github.com/nlnwa/veidemann-api/go/frontier/v1"
	"github.com/nlnwa/veidemann-api/go/scopechecker/v1"
	"testing"
)

var result *scopechecker.ScopeCheckResponse

func BenchmarkParse(b *testing.B) {
	server := &ScopeCheckerService{}
	qUri := &frontier.QueuedUri{
		Uri:           "http://foo.bar/aa bb/cc?jsessionid=1&foo#bar",
		SeedUri:       "http://foo.bar",
		Ip:            "127.0.0.1",
		DiscoveryPath: "RL",
		Referrer:      "http://foo.bar/",
		Annotation: []*config.Annotation{
			{Key: "testValue", Value: "True"},
		},
	}

	tests := []struct {
		name   string
		script string
		qUri   *frontier.QueuedUri
	}{
		{"1", "test(True).then(ChaffDetection)", qUri},
		{"2", "test(param(\"testValue\")).then(ChaffDetection).abort()", qUri},
		{"3", `
isSameHost().then(ChaffDetection)
isScheme('ftp').then(Blocked).abort()
maxHopsFromSeed(1).then(Include).abort()
`, qUri},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			request := &scopechecker.ScopeCheckRequest{
				QueuedUri:       tt.qUri,
				ScopeScriptName: "scope_script",
				ScopeScript:     tt.script,
			}

			for i := 0; i < b.N; i++ {
				got, err := server.ScopeCheck(context.TODO(), request)
				if err == nil {
					result = got
					if got.Error != nil {
						b.Error(got)
					}
				}
			}
		})
	}
}
