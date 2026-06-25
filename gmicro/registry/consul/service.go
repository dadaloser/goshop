package consul

import (
	"sync"
	"sync/atomic"

	"goshop/gmicro/registry"
)

type serviceSet struct {
	serviceName string
	watcher     map[*watcher]struct{}
	services    *atomic.Value
	lock        sync.RWMutex
}

func (s *serviceSet) broadcast(ss []*registry.ServiceInstance) {
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
