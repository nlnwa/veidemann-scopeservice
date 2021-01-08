package script

import (
	"errors"
	"github.com/nlnwa/veidemann-api/go/commons/v1"
	"github.com/nlnwa/veidemann-api/go/frontier/v1"
	"github.com/nlnwa/veidemann-api/go/scopechecker/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"os"
	"strings"
	"time"
	"veidemann-scopeservice/pkg/telemetry"
)

const (
	urlKey        = "url"
	resultKey     = "result"
	debugKey      = "debug"
	stacktraceKey = "stacktrace"
)

var EndOfComputation = errors.New("end of computation")

var scriptLogger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().
	Timestamp().Logger().Level(zerolog.DebugLevel)

// RunScopeScript runs the Scope checking script and returns the Scope status.
func RunScopeScript(name string, src interface{}, qUri *frontier.QueuedUri, debug bool) *scopechecker.ScopeCheckResponse {
	resolve.AllowNestedDef = true      // allow def statements within function bodies
	resolve.AllowLambda = true         // allow lambda expressions
	resolve.AllowFloat = true          // allow floating point literals, the 'float' built-in, and x / y
	resolve.AllowSet = true            // allow the 'set' built-in
	resolve.AllowGlobalReassign = true // allow reassignment to top-level names; also, allow if/for/while at top-level
	resolve.AllowRecursion = true      // allow while statements and recursive functions

	consoleLog := strings.Builder{}

	// Parse input URI
	qUrl, err := Url(qUri)
	if err != nil {
		return &scopechecker.ScopeCheckResponse{
			Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
			ExcludeReason:   IllegalUri.AsInt32(),
			IncludeCheckUri: &commons.ParsedUri{Href: qUri.Uri},
			Error: &commons.Error{
				Code:   IllegalUri.AsInt32(),
				Msg:    "error parsing uri",
				Detail: err.Error(),
			},
			Console: consoleLog.String(),
		}
	}

	includeCheckUri := qUrl.AsCommonsParsedUri()

	// Compile source
	t := prometheus.NewTimer(telemetry.CompileScriptSeconds)
	_, prog, err := starlark.SourceProgram(name, src, starlark.StringDict{}.Has)
	if err != nil {
		return &scopechecker.ScopeCheckResponse{
			Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
			ExcludeReason:   RuntimeException.AsInt32(),
			IncludeCheckUri: includeCheckUri,
			Error: &commons.Error{
				Code:   RuntimeException.AsInt32(),
				Msg:    "error parsing scope script",
				Detail: err.Error(),
			},
			Console: consoleLog.String(),
		}
	}
	t.ObserveDuration()

	// The Thread defines the behavior of the built-in 'print' function.
	thread := &starlark.Thread{
		Name: "scope",
		Print: func(thread *starlark.Thread, msg string) {
			if thread.CallStackDepth() > 1 {
				consoleLog.WriteString(thread.CallFrame(1).Pos.String() + " " + msg + "\n")
				scriptLogger.Debug().Msg(thread.CallFrame(1).Pos.String() + " " + msg)
			} else {
				consoleLog.WriteString(msg + "\n")
				scriptLogger.Debug().Msg(msg)
			}
		},
	}

	// Set local variables
	thread.SetLocal(urlKey, qUrl)
	parameters := qUri.Annotation
	for _, a := range parameters {
		thread.SetLocal(a.Key, starlark.String(a.Value))
	}
	thread.SetLocal(debugKey, starlark.Bool(debug))

	// Execute script.
	t = prometheus.NewTimer(telemetry.CompileScriptSeconds)
	_, err = prog.Init(thread, nil)
	t.ObserveDuration()
	if err != nil {
		if evalErr, ok := err.(*starlark.EvalError); ok {
			if evalErr.Unwrap() == EndOfComputation {
				//	Computation was aborted
			} else {
				if w, ok := evalErr.Unwrap().(*wrappedError); ok {
					// Script returned Status wrapped as Error
					e := (*commons.Error)(w)
					return &scopechecker.ScopeCheckResponse{
						Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
						ExcludeReason:   e.Code,
						IncludeCheckUri: includeCheckUri,
						Error:           e,
						Console:         consoleLog.String(),
					}
				} else {
					return &scopechecker.ScopeCheckResponse{
						Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
						ExcludeReason:   RuntimeException.AsInt32(),
						IncludeCheckUri: includeCheckUri,
						Error: &commons.Error{
							Code:   RuntimeException.AsInt32(),
							Msg:    "error executing scope script",
							Detail: evalErr.Backtrace(),
						},
						Console: consoleLog.String(),
					}
				}
			}
		} else {
			return &scopechecker.ScopeCheckResponse{
				Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
				ExcludeReason:   RuntimeException.AsInt32(),
				IncludeCheckUri: includeCheckUri,
				Error: &commons.Error{
					Code:   RuntimeException.AsInt32(),
					Msg:    "unknown error executing scope script",
					Detail: err.Error(),
				},
				Console: consoleLog.String(),
			}
		}
	}

	s, ok := thread.Local(resultKey).(Status)
	if ok {
		if s == 0 {
			return &scopechecker.ScopeCheckResponse{
				Evaluation:      scopechecker.ScopeCheckResponse_INCLUDE,
				IncludeCheckUri: includeCheckUri,
				Console:         consoleLog.String(),
			}
		} else {
			return &scopechecker.ScopeCheckResponse{
				Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
				ExcludeReason:   s.AsInt32(),
				IncludeCheckUri: includeCheckUri,
				Console:         consoleLog.String(),
			}
		}
	} else {
		return &scopechecker.ScopeCheckResponse{
			Evaluation:      scopechecker.ScopeCheckResponse_EXCLUDE,
			ExcludeReason:   Blocked.AsInt32(),
			IncludeCheckUri: includeCheckUri,
			Error:           (*commons.Error)(Blocked.asError("No scope rules matched")),
			Console:         consoleLog.String(),
		}
	}
}
