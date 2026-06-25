package main

import (
	"fmt"
	"sync"
	"time"
)

// --- 1. 定义观察者接口 ---
// 任何想要监听变化的结构体，都必须实现这个接口
type Observer interface {
	Update(data string)
}

// --- 2. 定义被观察者 (Subject) ---
type ConfigCenter struct {
	mu        sync.RWMutex
	data      string              // 当前的配置数据
	observers map[string]Observer // 存储所有观察者 (key为观察者名称)
}

// 创建一个新的配置中心
func NewConfigCenter(initialData string) *ConfigCenter {
	return &ConfigCenter{
		data:      initialData,
		observers: make(map[string]Observer),
	}
}

// 注册观察者
func (c *ConfigCenter) Attach(name string, obs Observer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.observers[name] = obs
	fmt.Printf("📢 观察者 [%s] 已加入监听\n", name)
}

// 移除观察者
func (c *ConfigCenter) Detach(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.observers, name)
	fmt.Printf("👋 观察者 [%s] 已停止监听\n", name)
}

// 更新数据并通知所有观察者 (广播)
func (c *ConfigCenter) SetData(newData string) {
	c.mu.Lock()
	c.data = newData
	c.mu.Unlock()

	fmt.Printf("\n🔥 数据已更新为: [%s]，正在广播通知...\n", newData)

	// 遍历并通知 (实际项目中这里可能需要加锁或使用 sync.Map)
	c.mu.RLock()
	for _, obs := range c.observers {
		// 在实际的长轮询场景中，这里通常是唤醒一个 channel
		obs.Update(c.data)
	}
	c.mu.RUnlock()
}

// 获取当前数据
func (c *ConfigCenter) GetData() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data
}

// --- 3. 定义具体的观察者 (Client) ---
type AppConfig struct {
	Name string
	// 模拟长轮询中的“事件通道”
	// 当服务端有更新时，会向这个通道发送信号
	eventChan    chan struct{}
	configCenter *ConfigCenter // 持有被观察者的引用，以便拉取数据
}

func NewAppConfig(name string, center *ConfigCenter) *AppConfig {
	ac := &AppConfig{
		Name: name,
		// 带缓冲的通道，防止多次变更导致阻塞（或者配合 select default 使用）
		eventChan:    make(chan struct{}, 1),
		configCenter: center,
	}
	// 将自己注册到配置中心
	center.Attach(name, ac)
	return ac
}

// 实现 Observer 接口
// 这个方法由“配置中心”调用，告诉 App：“嘿，有更新了”
func (ac *AppConfig) Update(data string) {
	// 非阻塞发送信号，模拟长轮询的响应返回
	select {
	case ac.eventChan <- struct{}{}:
		// 信号发送成功
	default:
		// 通道满了，说明上一次的通知还没处理完，丢弃本次（或者覆盖）
	}
}

// 模拟业务协程：持续监听配置变化 (长轮询的客户端逻辑)
func (ac *AppConfig) StartWatching() {
	fmt.Printf("👀 [%s] 开始监听配置变化...\n", ac.Name)

	// 模拟初始拉取
	currentData := ac.configCenter.GetData()
	fmt.Printf("   -> [%s] 初始配置: %s\n", ac.Name, currentData)

	for {
		// 1. 阻塞等待信号 (这就相当于 HTTP 请求挂起)
		<-ac.eventChan

		// 2. 收到信号后，立即去拉取最新数据
		newData := ac.configCenter.GetData()
		fmt.Printf("   ✅ [%s] 收到变更通知，新配置: %s\n", ac.Name, newData)

		// 这里可以加入业务逻辑，比如重新加载数据库连接等
	}
}

// --- 4. 主函数演示 ---
func main() {
	// 1. 初始化配置中心
	center := NewConfigCenter("v1.0.0")

	// 2. 创建两个观察者（比如两个微服务实例）
	serviceA := NewAppConfig("服务A", center)
	serviceB := NewAppConfig("服务B", center)

	// 3. 启动观察者的监听协程
	go serviceA.StartWatching()
	go serviceB.StartWatching()

	// 让主协程等待一会，确保观察者启动
	time.Sleep(1 * time.Second)

	// 4. 模拟配置变更
	time.Sleep(2 * time.Second)
	center.SetData("v1.1.0") // 触发通知

	time.Sleep(2 * time.Second)
	center.SetData("v1.2.0-hotfix") // 再次触发通知

	// 5. 演示移除观察者
	center.Detach("服务A")
	time.Sleep(1 * time.Second)
	center.SetData("v2.0.0") // 只有服务B会收到通知

	// 防止主程序退出（实际项目中通常是死循环或信号监听）
	time.Sleep(2 * time.Second)
}
