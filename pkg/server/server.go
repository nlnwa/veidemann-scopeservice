package server

import (
	"context"
	"fmt"
	"github.com/nlnwa/veidemann-api/go/frontier/v1"
	"github.com/nlnwa/veidemann-api/go/scopechecker/v1"
	"github.com/nlnwa/veidemann-api/go/uricanonicalizer/v1"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"veidemann-scopeservice/pkg/script"
	"veidemann-scopeservice/pkg/telemetry"
)

type GrpcServer struct {
	listenHost string
	listenPort int
	grpcServer *grpc.Server
}

func New(host string, port int) *GrpcServer {
	s := &GrpcServer{
		listenHost: host,
		listenPort: port,
	}
	return s
}

func (s *GrpcServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.listenHost, s.listenPort))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	tracer := opentracing.GlobalTracer()
	var opts = []grpc.ServerOption{
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	}
	s.grpcServer = grpc.NewServer(opts...)
	scopechecker.RegisterScopesCheckerServiceServer(s.grpcServer, &ScopeCheckerService{})
	uricanonicalizer.RegisterUriCanonicalizerServiceServer(s.grpcServer, &UriCanonicalizerService{})

	log.Info().Msgf("Scope Service listening on %s", lis.Addr())
	return s.grpcServer.Serve(lis)
}

func (s *GrpcServer) Shutdown() {
	log.Info().Msg("Shutting down Scope Service")
	s.grpcServer.GracefulStop()
}

type ScopeCheckerService struct {
	scopechecker.UnimplementedScopesCheckerServiceServer
}

func (s *ScopeCheckerService) ScopeCheck(_ context.Context, request *scopechecker.ScopeCheckRequest) (*scopechecker.ScopeCheckResponse, error) {
	telemetry.ScopechecksTotal.Inc()
	result := script.RunScopeScript("scope_script", request.ScopeScript, request.QueuedUri, request.Debug)
	telemetry.ScopecheckResponseTotal.With(prometheus.Labels{"code": strconv.Itoa(int(result.ExcludeReason))}).Inc()
	return result, nil
}

type UriCanonicalizerService struct {
	uricanonicalizer.UnimplementedUriCanonicalizerServiceServer
}

func (u *UriCanonicalizerService) Canonicalize(_ context.Context, uri *frontier.QueuedUri) (*frontier.QueuedUri, error) {
	telemetry.CanonicalizationsTotal.Inc()
	canonicalized, err := script.CrawlCanonicalizationProfile.Parse(uri.Uri)
	if err == nil {
		uri.Uri = canonicalized.String()
		return uri, nil
	}
	return uri, err
}
