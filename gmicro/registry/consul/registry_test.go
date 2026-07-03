package consul

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"goshop/gmicro/registry"
)

func skipIfConsulUnavailable(t *testing.T) {
	t.Helper()

	conn, err := net.DialTimeout("tcp", "127.0.0.1:8500", 200*time.Millisecond)
	if err != nil {
		t.Skipf("consul is not available at 127.0.0.1:8500: %v", err)
	}
	_ = conn.Close()
}

func tcpServer(t *testing.T, lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		fmt.Println("get tcp")
		conn.Close()
	}
}

func TestRegistry_Register(t *testing.T) {
	skipIfConsulUnavailable(t)

	opts := []Option{
		WithHealthCheck(false),
	}

	type args struct {
		ctx        context.Context
		serverName string
		server     []*registry.ServiceInstance
	}

	test := []struct {
		name    string
		args    args
		want    []*registry.ServiceInstance
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				ctx:        context.Background(),
				serverName: "server-1",
				server: []*registry.ServiceInstance{
					{
						ID:        "1",
						Name:      "server-1",
						Version:   "v0.0.1",
						Metadata:  nil,
						Endpoints: []string{"http://127.0.0.1:8000"},
					},
				},
			},
			want: []*registry.ServiceInstance{
				{
					ID:        "1",
					Name:      "server-1",
					Version:   "v0.0.1",
					Metadata:  nil,
					Endpoints: []string{"http://127.0.0.1:8000"},
				},
			},
			wantErr: false,
		},
		{
			name: "registry new service replace old service",
			args: args{
				ctx:        context.Background(),
				serverName: "server-1",
				server: []*registry.ServiceInstance{
					{
						ID:        "1",
						Name:      "server-1",
						Version:   "v0.0.1",
						Metadata:  nil,
						Endpoints: []string{"http://127.0.0.1:8000"},
					},
					{
						ID:        "1",
						Name:      "server-1",
						Version:   "v0.0.2",
						Metadata:  nil,
						Endpoints: []string{"http://127.0.0.1:8000"},
					},
				},
			},
			want: []*registry.ServiceInstance{
				{
					ID:        "1",
					Name:      "server-1",
					Version:   "v0.0.2",
					Metadata:  nil,
					Endpoints: []string{"http://127.0.0.1:8000"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			cli, err := api.NewClient(&api.Config{Address: "127.0.0.1:8500"})
			if err != nil {
				t.Fatalf("create consul client failed: %v", err)
			}

			r := New(cli, opts...)

			for _, instance := range tt.args.server {
				err = r.Register(tt.args.ctx, instance)
				if err != nil {
					t.Error(err)
				}
			}

			watch, err := r.Watch(tt.args.ctx, tt.args.serverName)
			if err != nil {
				t.Error(err)
			}
			got, err := watch.Next()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("GetService() got = %v", got)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetService() got = %v, want %v", got, tt.want)
			}

			for _, instance := range tt.args.server {
				_ = r.Deregister(tt.args.ctx, instance)
			}
		})
	}
}

func TestRegistry_GetService(t *testing.T) {
	skipIfConsulUnavailable(t)

	addr := fmt.Sprintf("%s:9091", getIntranetIP())
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Errorf("listen tcp %s failed!", addr)
		t.Fail()
	}
	defer lis.Close()
	go tcpServer(t, lis)
	time.Sleep(time.Millisecond * 100)
	cli, err := api.NewClient(&api.Config{Address: "127.0.0.1:8500"})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	opts := []Option{
		WithHeartbeat(true),
		WithHealthCheck(true),
		WithHealthCheckInterval(5),
	}
	r := New(cli, opts...)

	instance1 := &registry.ServiceInstance{
		ID:        "1",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	type fields struct {
		registry *Registry
	}
	type args struct {
		ctx         context.Context
		serviceName string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      []*registry.ServiceInstance
		wantErr   bool
		preFunc   func(t *testing.T)
		deferFunc func(t *testing.T)
	}{
		{
			name:   "normal",
			fields: fields{r},
			args: args{
				ctx:         context.Background(),
				serviceName: "server-1",
			},
			want:    []*registry.ServiceInstance{instance1},
			wantErr: false,
			preFunc: func(t *testing.T) {
				if err := r.Register(context.Background(), instance1); err != nil {
					t.Error(err)
				}
				watch, err := r.Watch(context.Background(), instance1.Name)
				if err != nil {
					t.Error(err)
				}
				_, err = watch.Next()
				if err != nil {
					t.Error(err)
				}
			},
			deferFunc: func(t *testing.T) {
				err := r.Deregister(context.Background(), instance1)
				if err != nil {
					t.Error(err)
				}
			},
		},
		{
			name:   "can't get any",
			fields: fields{r},
			args: args{
				ctx:         context.Background(),
				serviceName: "server-x",
			},
			want:    nil,
			wantErr: true,
			preFunc: func(t *testing.T) {
				if err := r.Register(context.Background(), instance1); err != nil {
					t.Error(err)
				}
				watch, err := r.Watch(context.Background(), instance1.Name)
				if err != nil {
					t.Error(err)
				}
				_, err = watch.Next()
				if err != nil {
					t.Error(err)
				}
			},
			deferFunc: func(t *testing.T) {
				err := r.Deregister(context.Background(), instance1)
				if err != nil {
					t.Error(err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.preFunc != nil {
				test.preFunc(t)
			}
			if test.deferFunc != nil {
				defer test.deferFunc(t)
			}

			service, err := test.fields.registry.GetService(context.Background(), test.args.serviceName)
			if (err != nil) != test.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, test.wantErr)
				t.Errorf("GetService() got = %v", service)
				return
			}
			if !reflect.DeepEqual(service, test.want) {
				t.Errorf("GetService() got = %v, want %v", service, test.want)
			}
		})
	}
}

func TestRegistry_Watch(t *testing.T) {
	skipIfConsulUnavailable(t)

	addr := fmt.Sprintf("%s:9091", getIntranetIP())

	time.Sleep(time.Millisecond * 100)
	cli, err := api.NewClient(&api.Config{Address: "127.0.0.1:8500", WaitTime: 2 * time.Second})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}

	instance1 := &registry.ServiceInstance{
		ID:        "1",
		Name:      "server-1",
		Version:   "v0.0.1",
		Endpoints: []string{fmt.Sprintf("tcp://%s?isSecure=false", addr)},
	}

	type args struct {
		ctx      context.Context
		opts     []Option
		instance *registry.ServiceInstance
	}
	tests := []struct {
		name    string
		args    args
		want    []*registry.ServiceInstance
		wantErr bool
		preFunc func(t *testing.T)
	}{
		{
			name: "normal",
			args: args{
				ctx:      context.Background(),
				instance: instance1,
				opts: []Option{
					WithHealthCheck(false),
				},
			},
			want:    []*registry.ServiceInstance{instance1},
			wantErr: false,
			preFunc: func(t *testing.T) {
			},
		},
		{
			name: "register with healthCheck",
			args: args{
				ctx:      context.Background(),
				instance: instance1,
				opts: []Option{
					WithHeartbeat(true),
					WithHealthCheck(true),
					WithHealthCheckInterval(5),
				},
			},
			want:    []*registry.ServiceInstance{instance1},
			wantErr: false,
			preFunc: func(t *testing.T) {
				lis, err := net.Listen("tcp", addr)
				if err != nil {
					t.Errorf("listen tcp %s failed!", addr)
				}
				go tcpServer(t, lis)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.preFunc != nil {
				tt.preFunc(t)
			}

			r := New(cli, tt.args.opts...)

			err := r.Register(tt.args.ctx, tt.args.instance)
			if err != nil {
				t.Error(err)
			}
			defer func() {
				err = r.Deregister(tt.args.ctx, tt.args.instance)
				if err != nil {
					t.Error(err)
				}
			}()

			watch, err := r.Watch(tt.args.ctx, tt.args.instance.Name)
			if err != nil {
				t.Error(err)
			}

			service, err := watch.Next()

			if (err != nil) != tt.wantErr {
				t.Errorf("GetService() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("GetService() got = %v", service)
				return
			}
			if !reflect.DeepEqual(service, tt.want) {
				t.Errorf("GetService() got = %v, want %v", service, tt.want)
			}
		})
	}
}

func getIntranetIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func TestServiceSetBroadcastsEmptyServiceList(t *testing.T) {
	set := &serviceSet{
		watcher:  make(map[*watcher]struct{}),
		services: &atomic.Value{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &watcher{
		event:  make(chan struct{}, 1),
		set:    set,
		ctx:    ctx,
		cancel: cancel,
	}
	set.watcher[w] = struct{}{}

	set.broadcast([]*registry.ServiceInstance{{ID: "1", Name: "server-1"}})
	got, err := w.Next()
	if err != nil {
		t.Fatalf("Next() error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("Next() services = %v, want one service", got)
	}

	set.broadcast(nil)
	got, err = w.Next()
	if err != nil {
		t.Fatalf("Next() error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("Next() services = %v, want empty service list", got)
	}
}

func TestRegistryWatchContinuesAfterInitialResolve(t *testing.T) {
	var calls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health/service/server-1" {
			t.Fatalf("request path = %s, want /v1/health/service/server-1", r.URL.Path)
		}
		call := atomic.AddInt64(&calls, 1)
		if call == 1 {
			w.Header().Set("X-Consul-Index", "1")
			_ = json.NewEncoder(w).Encode([]*api.ServiceEntry{
				{
					Service: &api.AgentService{
						ID:      "1",
						Service: "server-1",
						Tags:    []string{"version=v1"},
						TaggedAddresses: map[string]api.ServiceAddress{
							"grpc": {
								Address: "grpc://127.0.0.1:9000",
								Port:    9000,
							},
						},
					},
				},
			})
			return
		}

		w.Header().Set("X-Consul-Index", "2")
		_ = json.NewEncoder(w).Encode([]*api.ServiceEntry{})
	}))
	t.Cleanup(server.Close)

	cli, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	r := New(cli)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	watch, err := r.Watch(ctx, "server-1")
	if err != nil {
		t.Fatalf("Watch() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = watch.Stop()
	})

	got, err := watch.Next()
	if err != nil {
		t.Fatalf("Next() initial error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("Next() initial services = %v, want one service", got)
	}

	got, err = watch.Next()
	if err != nil {
		t.Fatalf("Next() update error = %v, want nil", err)
	}
	if len(got) != 0 {
		t.Fatalf("Next() update services = %v, want empty service list", got)
	}
}

func TestRegistryWatchRetriesAfterInitialResolveError(t *testing.T) {
	var calls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt64(&calls, 1)
		if call == 1 {
			http.Error(w, "consul unavailable", http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-Consul-Index", fmt.Sprintf("%d", call))
		_ = json.NewEncoder(w).Encode([]*api.ServiceEntry{
			{
				Service: &api.AgentService{
					ID:      "1",
					Service: "server-1",
					Tags:    []string{"version=v1"},
					TaggedAddresses: map[string]api.ServiceAddress{
						"grpc": {
							Address: "grpc://127.0.0.1:9000",
							Port:    9000,
						},
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	cli, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	r := New(cli)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	watch, err := r.Watch(ctx, "server-1")
	if err != nil {
		t.Fatalf("Watch() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = watch.Stop()
	})

	got, err := watch.Next()
	if err != nil {
		t.Fatalf("Next() error = %v, want nil after retry", err)
	}
	if len(got) != 1 {
		t.Fatalf("Next() services = %v, want one service after retry", got)
	}
	if atomic.LoadInt64(&calls) < 2 {
		t.Fatalf("consul service calls = %d, want retry", calls)
	}
}

func TestRegistryWatchRestartsResolverAfterFirstWatcherStops(t *testing.T) {
	var calls int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt64(&calls, 1)
		w.Header().Set("X-Consul-Index", fmt.Sprintf("%d", call))
		_ = json.NewEncoder(w).Encode([]*api.ServiceEntry{
			{
				Service: &api.AgentService{
					ID:      fmt.Sprintf("%d", call),
					Service: "server-1",
					Tags:    []string{"version=v1"},
					TaggedAddresses: map[string]api.ServiceAddress{
						"grpc": {
							Address: fmt.Sprintf("grpc://127.0.0.1:%d", 9000+call),
							Port:    int(9000 + call),
						},
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	cli, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	r := New(cli)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	first, err := r.Watch(ctx, "server-1")
	if err != nil {
		t.Fatalf("first Watch() error = %v, want nil", err)
	}
	if _, err := first.Next(); err != nil {
		t.Fatalf("first Next() error = %v, want nil", err)
	}
	if err := first.Stop(); err != nil {
		t.Fatalf("first Stop() error = %v, want nil", err)
	}

	second, err := r.Watch(ctx, "server-1")
	if err != nil {
		t.Fatalf("second Watch() error = %v, want nil", err)
	}
	t.Cleanup(func() {
		_ = second.Stop()
	})
	got, err := second.Next()
	if err != nil {
		t.Fatalf("second Next() error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("second Next() services = %v, want one service", got)
	}
	deadline := time.After(time.Second)
	for atomic.LoadInt64(&calls) < 2 {
		select {
		case <-deadline:
			t.Fatalf("consul service calls = %d, want resolver restarted for second watcher", calls)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestWatcherStopCancelsResolverAfterLastWatcher(t *testing.T) {
	enteredLongPoll := make(chan struct{})
	requestDone := make(chan struct{})
	var longPollStarted atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !longPollStarted.CompareAndSwap(false, true) {
			close(enteredLongPoll)
			<-r.Context().Done()
			close(requestDone)
			return
		}

		w.Header().Set("X-Consul-Index", "1")
		_ = json.NewEncoder(w).Encode([]*api.ServiceEntry{
			{
				Service: &api.AgentService{
					ID:      "1",
					Service: "server-1",
					Tags:    []string{"version=v1"},
					TaggedAddresses: map[string]api.ServiceAddress{
						"grpc": {
							Address: "grpc://127.0.0.1:9000",
							Port:    9000,
						},
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	cli, err := api.NewClient(&api.Config{Address: server.URL})
	if err != nil {
		t.Fatalf("create consul client failed: %v", err)
	}
	r := New(cli)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	watch, err := r.Watch(ctx, "server-1")
	if err != nil {
		t.Fatalf("Watch() error = %v, want nil", err)
	}
	if _, err := watch.Next(); err != nil {
		t.Fatalf("Next() error = %v, want nil", err)
	}

	select {
	case <-enteredLongPoll:
	case <-time.After(time.Second):
		t.Fatal("resolver did not enter long poll")
	}
	if err := watch.Stop(); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("resolver long poll was not canceled after last watcher stopped")
	}
}
