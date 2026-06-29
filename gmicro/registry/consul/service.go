package consul

import (
	"context"
	"sync"
	"sync/atomic"

	"goshop/gmicro/registry"
)

type serviceSet struct {
	serviceName string
	watcher     map[*watcher]struct{}
	services    *atomic.Value
	lock        sync.RWMutex

	resolverCtx     context.Context
	resolverCancel  context.CancelFunc
	resolverRunning bool
}

func (s *serviceSet) broadcast(ss []*registry.ServiceInstance) {
	if ss == nil {
		ss = []*registry.ServiceInstance{}
	}
	//原子操作， 保证线程安全， 我们平时写struct的时候
	s.services.Store(ss)
	s.lock.RLock()
	defer s.lock.RUnlock()
	// 使用非阻塞发送，防止慢速消费者阻塞整个广播流程
	for k := range s.watcher {
		select {
		case k.event <- struct{}{}:
		default:
			// 通道已满，丢弃本次通知。
			// 注意：这可能导致消费者短暂持有旧数据，但在服务发现场景中通常可接受
			// 也可以在此处记录日志：log.Printf("watcher event channel full, dropping update")
		}
	}
}

func (s *serviceSet) startResolver(parent context.Context, resolve func(context.Context, *serviceSet) error) error {
	s.lock.Lock()
	if s.resolverRunning {
		s.lock.Unlock()
		return nil
	}
	base := context.Background()
	if parent != nil {
		base = context.WithoutCancel(parent)
	}
	ctx, cancel := context.WithCancel(base)
	s.resolverCtx = ctx
	s.resolverCancel = cancel
	s.resolverRunning = true
	s.lock.Unlock()

	if err := resolve(ctx, s); err != nil {
		cancel()
		s.lock.Lock()
		if s.resolverCtx == ctx {
			s.resolverCtx = nil
			s.resolverCancel = nil
			s.resolverRunning = false
		}
		s.lock.Unlock()
		return err
	}
	return nil
}

func (s *serviceSet) addWatcher(w *watcher) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.watcher[w] = struct{}{}
}

func (s *serviceSet) removeWatcher(w *watcher) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.watcher, w)
	if len(s.watcher) == 0 && s.resolverCancel != nil {
		s.resolverCancel()
		s.resolverCancel = nil
		s.resolverCtx = nil
		s.resolverRunning = false
	}
}

func (s *serviceSet) resolverStopped(ctx context.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.resolverCtx == ctx {
		s.resolverCancel = nil
		s.resolverCtx = nil
		s.resolverRunning = false
	}
}
