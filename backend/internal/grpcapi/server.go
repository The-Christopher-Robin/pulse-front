package grpcapi

import (
	"context"
	"errors"
	"net"
	"time"

	"google.golang.org/grpc"

	"github.com/The-Christopher-Robin/pulse-front/backend/internal/analytics"
	"github.com/The-Christopher-Robin/pulse-front/backend/internal/experiments"
	pb "github.com/The-Christopher-Robin/pulse-front/backend/internal/grpcapi/pb"
)

type telemetryServer struct {
	pb.UnimplementedTelemetryServiceServer
	writer *analytics.Writer
}

func (s *telemetryServer) RecordExposure(ctx context.Context, req *pb.ExposureRequest) (*pb.ExposureResponse, error) {
	if req.GetExperimentKey() == "" || req.GetVariantKey() == "" || req.GetUserId() == "" {
		return nil, errors.New("experiment_key, variant_key, user_id required")
	}
	ts := req.GetTimestampMs()
	occurred := time.Now().UTC()
	if ts > 0 {
		occurred = time.UnixMilli(ts).UTC()
	}
	s.writer.Enqueue(experiments.Assignment{
		ExperimentKey: req.GetExperimentKey(),
		VariantKey:    req.GetVariantKey(),
		UserID:        req.GetUserId(),
		OccurredAt:    occurred,
		Exposed:       true,
	})
	return &pb.ExposureResponse{Accepted: true}, nil
}

func (s *telemetryServer) RecordEvent(ctx context.Context, req *pb.EventRequest) (*pb.EventResponse, error) {
	if req.GetUserId() == "" || req.GetEventType() == "" {
		return nil, errors.New("user_id and event_type required")
	}
	props := map[string]interface{}{}
	for k, v := range req.GetProperties() {
		props[k] = v
	}
	ts := req.GetTimestampMs()
	occurred := time.Now().UTC()
	if ts > 0 {
		occurred = time.UnixMilli(ts).UTC()
	}
	err := s.writer.TrackEvent(analytics.Event{
		UserID:     req.GetUserId(),
		EventType:  req.GetEventType(),
		TargetID:   req.GetTargetId(),
		Properties: props,
		OccurredAt: occurred,
	})
	if err != nil {
		return &pb.EventResponse{Accepted: false}, err
	}
	return &pb.EventResponse{Accepted: true}, nil
}

type Server struct {
	listener net.Listener
	grpc     *grpc.Server
}

func NewServer(addr string, writer *analytics.Writer) (*Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	gs := grpc.NewServer()
	pb.RegisterTelemetryServiceServer(gs, &telemetryServer{writer: writer})
	return &Server{listener: lis, grpc: gs}, nil
}

func (s *Server) Start() error {
	return s.grpc.Serve(s.listener)
}

func (s *Server) GracefulStop() {
	s.grpc.GracefulStop()
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}
