/*
 * Copyright 2020 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package telemetry

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-client-go/log"
	"io"
)

// Init returns an instance of Jaeger Tracer that samples 100% of traces and logs all spans to stdout.
func InitTracer(service string) (opentracing.Tracer, io.Closer) {
	cfg, err := config.FromEnv()
	if err != nil {
		log.StdLogger.Infof("ERROR: cannot init Jaeger from environment: %v", err)
		return nil, nil
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = service
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(log.StdLogger))
	if err != nil {
		log.StdLogger.Infof("ERROR: cannot init Jaeger: %v", err)
		return nil, nil
	}
	return tracer, closer
}
