package app

import (
	"context"
	"fmt"
	"net/url"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"goshop/gmicro/core/trace"
	"goshop/gmicro/registry"
	gs "goshop/gmicro/server"
	"goshop/pkg/log"
	"os"
	"os/signal"
	"sync"
)

type App struct {
	opts options

	lk       sync.Mutex
	instance *registry.ServiceInstance

	cancel func()
}

type readyServer interface {
	Ready() <-chan struct{}
}

type endpointServer interface {
	Endpoint() *url.URL
}

func (a *App) servers() []gs.Server {
	servers := make([]gs.Server, 0, len(a.opts.servers)+2)
	servers = append(servers, a.opts.servers...)
	if a.opts.restServer != nil {
		servers = append(servers, a.opts.restServer)
	}
	if a.opts.rpcServer != nil {
		servers = append(servers, a.opts.rpcServer)
	}
	return servers
}

func New(opts ...Option) *App {
	o := options{
		sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT},
		registrarTimeout: 10 * time.Second,
		stopTimeout:      10 * time.Second,
	}

	if id, err := uuid.NewUUID(); err == nil {
		o.id = id.String()
	}

	for _, opt := range opts {
		opt(&o)
	}

	return &App{
		opts: o,
	}
}

// 启动整个服务
func (a *App) Run() error {
	servers := a.servers()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	eg, ctx := errgroup.WithContext(ctx)
	wg := sync.WaitGroup{}
	readyChans := make([]<-chan struct{}, 0, len(servers))
	for _, srv := range servers {
		if readySrv, ok := srv.(readyServer); ok {
			readyChans = append(readyChans, readySrv.Ready())
		}
		//启动server
		//在启动一个goroutine 去监听是否有err产生
		srv := srv
		eg.Go(func() error {
			<-ctx.Done() //wait for stop signal
			//不可能无休止的等待stop
			sctx, cancel := context.WithTimeout(context.Background(), a.opts.stopTimeout)
			defer cancel()
			return srv.Stop(sctx)
		})

		wg.Add(1)
		eg.Go(func() error {
			wg.Done()
			log.Info("start rest server")
			return srv.Start(ctx)
		})
	}

	wg.Wait()

	//等实现了Ready()的server完成监听，再构建 service instance 并注册，避免服务还没起来就暴露到注册中心。
	if err := waitReady(ctx, readyChans, a.opts.registrarTimeout); err != nil {
		cancel()
		_ = eg.Wait()
		return err
	}

	//注册的信息
	instance, err := a.buildInstance()
	if err != nil {
		cancel()
		_ = eg.Wait()
		return err
	}

	//这个变量可能被其他的goroutine访问到
	a.lk.Lock()
	a.instance = instance
	a.lk.Unlock()

	//注册服务
	if a.opts.registrar != nil {
		rctx, rcancel := context.WithTimeout(context.Background(), a.opts.registrarTimeout)
		defer rcancel()
		err := a.opts.registrar.Register(rctx, instance)
		if err != nil {
			log.Errorf("register service error: %s", err)
			cancel()
			_ = eg.Wait()
			return err
		}
	}

	//监听退出信息
	c := make(chan os.Signal, 1)
	signal.Notify(c, a.opts.sigs...)
	eg.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c:
			return a.Stop()
		}
	})
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func waitReady(ctx context.Context, readyChans []<-chan struct{}, timeout time.Duration) error {
	if len(readyChans) == 0 {
		return nil
	}

	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for _, ready := range readyChans {
		select {
		case <-ready:
		case <-tctx.Done():
			return fmt.Errorf("server startup wait failed: %w", tctx.Err())
		}
	}
	return nil
}

/*
http basic 认证
cache： 1. redis 2. memcache 3. local cache
jwt
*/
// 停止服务
func (a *App) Stop() error {
	a.lk.Lock()
	instance := a.instance
	a.lk.Unlock()

	var stopErr error
	log.Info("start deregister service")
	if a.opts.registrar != nil && instance != nil {
		rctx, rcancel := context.WithTimeout(context.Background(), a.opts.stopTimeout)
		if err := a.opts.registrar.Deregister(rctx, instance); err != nil {
			log.Errorf("deregister service error: %s", err)
			stopErr = err
		}
		rcancel()
	}

	if a.cancel != nil {
		a.cancel()
	}

	tctx, tcancel := context.WithTimeout(context.Background(), a.opts.stopTimeout)
	defer tcancel()
	if err := trace.Shutdown(tctx); err != nil && stopErr == nil {
		stopErr = err
	}
	return stopErr
}

// 创建服务注册结构体
func (a *App) buildInstance() (*registry.ServiceInstance, error) {
	endpoints := make([]string, 0)
	for _, e := range a.opts.endpoints {
		endpoints = append(endpoints, e.String())
	}

	//从rpcserver， restserver去主动获取这些信息
	if a.opts.rpcServer != nil {
		if a.opts.rpcServer.Endpoint() != nil {
			endpoints = append(endpoints, a.opts.rpcServer.Endpoint().String())
		} else {
			u := &url.URL{
				Scheme: "grpc",
				Host:   a.opts.rpcServer.Address(),
			}
			endpoints = append(endpoints, u.String())
		}
	}
	if a.opts.restServer != nil {
		if e := a.opts.restServer.Endpoint(); e != nil {
			endpoints = append(endpoints, e.String())
		}
	}

	return &registry.ServiceInstance{
		ID:        a.opts.id,
		Name:      a.opts.name,
		Endpoints: endpoints,
	}, nil
}
