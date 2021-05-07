package server

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/backchannel"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/maintenance"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	gitalylog "gitlab.com/gitlab-org/gitaly/internal/log"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc"
)

// GitalyServerFactory is a factory of gitaly grpc servers
type GitalyServerFactory struct {
	registry *backchannel.Registry
	cfg      config.Cfg
	servers  []*grpc.Server
}

// NewGitalyServerFactory allows to create and start secure/insecure 'grpc.Server'-s with gitaly-ruby
// server shared in between.
func NewGitalyServerFactory(cfg config.Cfg, registry *backchannel.Registry) *GitalyServerFactory {
	return &GitalyServerFactory{cfg: cfg, registry: registry}
}

// StartWorkers will start any auxiliary background workers that are allowed
// to fail without stopping the rest of the server.
func (s *GitalyServerFactory) StartWorkers(ctx context.Context, l logrus.FieldLogger, cfg config.Cfg) (func(), error) {
	var opts []grpc.DialOption
	if cfg.Auth.Token != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(
			gitalyauth.RPCCredentialsV2(cfg.Auth.Token),
		))
	}

	cc, err := client.Dial("unix://"+cfg.GitalyInternalSocketPath(), opts)
	if err != nil {
		return nil, err
	}

	errQ := make(chan error)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		errQ <- maintenance.NewDailyWorker().StartDaily(
			ctx,
			l,
			cfg.DailyMaintenance,
			maintenance.OptimizeReposRandomly(
				cfg.Storages,
				gitalypb.NewRepositoryServiceClient(cc),
				helper.NewTimerTicker(1*time.Second),
				rand.New(rand.NewSource(time.Now().UnixNano())),
			),
		)
	}()

	shutdown := func() {
		cancel()

		// give the worker 5 seconds to shutdown gracefully
		timeout := 5 * time.Second

		var err error
		select {
		case err = <-errQ:
			break
		case <-time.After(timeout):
			err = fmt.Errorf("timed out after %s", timeout)
		}
		if err != nil && err != context.Canceled {
			l.WithError(err).Error("maintenance worker shutdown")
		}
	}

	return shutdown, nil
}

// Stop stops all servers started by calling Serve and the gitaly-ruby server.
func (s *GitalyServerFactory) Stop() {
	for _, srv := range s.servers {
		srv.Stop()
	}
}

// GracefulStop stops both the secure and insecure servers gracefully
func (s *GitalyServerFactory) GracefulStop() {
	wg := sync.WaitGroup{}

	for _, srv := range s.servers {
		wg.Add(1)

		go func(s *grpc.Server) {
			s.GracefulStop()
			wg.Done()
		}(srv)
	}

	wg.Wait()
}

// Create returns newly instantiated and initialized with interceptors instance of the gRPC server.
func (s *GitalyServerFactory) Create(secure bool) (*grpc.Server, error) {
	server, err := New(secure, s.cfg, gitalylog.Default(), s.registry)
	if err != nil {
		return nil, err
	}

	s.servers = append(s.servers, server)
	return server, nil
}
