package main

import (
	"context"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// TestDistributedLock 演示如何使用 etcd 的 concurrency 包实现分布式锁（高级封装方式）
// 优点：代码简洁，自动处理租约续约、锁等待、异常恢复
// 缺点：封装了内部细节，不利于理解底层实现原理
func TestDistributedLock(t *testing.T) {
	// 步骤 1: 创建 etcd 客户端
	// clientv3.New() 初始化一个 etcd v3 客户端连接
	client, err := clientv3.New(clientv3.Config{
		// Endpoints: etcd 集群的地址列表，这里连接本地单节点
		Endpoints: []string{"localhost:2379"},
		// DialTimeout: 连接超时时间，3秒内无法连接则报错
		DialTimeout: 3 * time.Second,
	})
	// 检查客户端是否创建成功
	if err != nil {
		// t.Fatalf: 测试失败并立即退出，打印错误信息
		t.Fatalf("Failed to connect to etcd: %v", err)
	}
	// defer 确保函数结束时关闭客户端连接，释放资源
	defer client.Close()

	// 步骤 2: 创建会话（Session）
	// Session 是 concurrency 包的核心概念，它内部封装了：
	//   - 租约（Lease）：自动创建一个 TTL=5秒 的租约
	//   - 租约续约（KeepAlive）：后台自动发送心跳保持租约活跃
	//   - 会话生命周期管理：Session 关闭时自动撤销租约
	session, err := concurrency.NewSession(
		client, // etcd 客户端
		// concurrency.WithTTL(5): 设置租约 TTL 为 5 秒
		// 如果进程崩溃或网络断开，5秒后租约自动过期，锁自动释放
		concurrency.WithTTL(5),
	)
	// 检查会话是否创建成功
	if err != nil {
		t.Fatalf("Failed to create etcd session: %v", err)
	}
	// defer 确保函数结束时关闭会话，撤销租约，释放锁
	defer session.Close()

	// 步骤 3: 创建分布式互斥锁（Mutex）
	// lockKey: 锁的唯一标识符，多个进程用相同的 key 竞争同一把锁
	lockKey := "my-distributed-lock"
	// concurrency.NewMutex 创建一个基于 session 的互斥锁对象
	// 内部实现：在 etcd 中创建一个前缀为 lockKey 的有序键（带租约）
	mutex := concurrency.NewMutex(session, lockKey)

	// 步骤 4: 获取锁（阻塞操作）
	// mutex.Lock() 尝试获取锁，如果锁已被其他进程持有，会阻塞等待
	// 工作原理：
	//   1. 在 etcd 中创建一个键：/my-distributed-lock/<lease_id>
	//   2. 检查自己创建的键是否是最小的（CreateRevision 最小）
	//   3. 如果是最小的，获取锁成功
	//   4. 如果不是，监听（Watch）前一个键的删除事件，等待锁释放
	if err := mutex.Lock(context.Background()); err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	// 获取锁成功后打印日志
	t.Logf("Lock acquired")

	// 步骤 5: 执行临界区代码（业务逻辑）
	// 这里用 Sleep 模拟耗时的业务处理，实际场景可能是：
	//   - 修改共享资源（数据库、配置）
	//   - 执行定时任务（防止多个实例重复执行）
	//   - 协调分布式操作
	time.Sleep(3 * time.Second)

	// 步骤 6: 释放锁
	// mutex.Unlock() 删除在 etcd 中创建的锁键，通知其他等待者
	if err := mutex.Unlock(context.Background()); err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}
	// 释放锁成功后打印日志
	t.Logf("Lock released")
}

// TestDistributedLockNormal 演示如何使用 etcd 原生 API 手动实现分布式锁（底层实现方式）
// 优点：完全掌控细节，理解分布式锁的实现原理
// 缺点：代码复杂，需要手动处理租约、事务、监听等逻辑
func TestDistributedLockNormal(t *testing.T) {
	// ==================== 步骤 1: 创建 etcd 客户端 ====================
	// 与 TestDistributedLock 相同，创建 etcd v3 客户端
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"}, // etcd 服务地址
		DialTimeout: 3 * time.Second,            // 连接超时时间
	})
	if err != nil {
		t.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer client.Close() // 函数结束时关闭客户端

	// 创建一个背景上下文，用于后续所有 etcd 操作
	// context.Background() 返回一个永不超时、永不取消的根上下文
	ctx := context.Background()

	// ==================== 步骤 2: 创建租约并启动自动续约 ====================
	// 租约（Lease）是 etcd 的核心机制，用于实现键的自动过期
	// client.Grant() 向 etcd 申请一个新租约
	leaseResp, err := client.Grant(
		ctx, // 上下文
		5,   // TTL（Time To Live）= 5秒，租约过期时间
	)
	if err != nil {
		t.Fatalf("Failed to create lease: %v", err)
	}
	// leaseResp.ID 是 etcd 分配的租约 ID，后续操作需要用到

	// 启动租约自动续约（KeepAlive）
	// 为什么需要续约？
	//   - 租约 TTL 只有 5 秒，如果不续约，5秒后租约过期，锁会自动释放
	//   - KeepAlive 会每隔 TTL/3 (约1.6秒) 发送一次心跳，保持租约活跃
	// client.KeepAlive() 启动后台续约，返回一个只读通道
	keepAliveChan, err := client.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		t.Fatalf("Failed to keep lease alive: %v", err)
	}

	// 启动一个 goroutine（后台线程）消费续约响应
	// 为什么需要消费？
	//   - 如果不从 keepAliveChan 读取数据，通道会满，导致续约阻塞
	//   - 即使不处理响应，也必须读取，保持通道流畅
	go func() {
		// range 循环持续读取通道中的续约响应
		// 每次续约成功，etcd 会发送一个 LeaseKeepAliveResponse
		for ka := range keepAliveChan {
			// ka.ID: 租约 ID
			// ka.TTL: 剩余生存时间（每次续约后重置为 5 秒）
			t.Logf("Received lease keep-alive response: %v", ka.ID)
		}
		// 通道关闭时（租约撤销或客户端关闭），循环自动退出
	}()

	// ==================== 步骤 3: 尝试获取锁（使用事务保证原子性）====================
getlock: // 定义一个标签，用于 goto 跳转重试
	// 锁的唯一标识符（所有竞争者使用相同的 key）
	lockKey := "my-distributed-lock-normal"

	// 创建一个事务（Transaction）
	// 事务保证操作的原子性：要么全部成功，要么全部失败
	txn := client.Txn(ctx)

	// 执行事务：If-Then-Else 结构
	txnResp, err := txn.
		// If 条件判断：检查 lockKey 是否存在
		If(
			// clientv3.Compare 创建一个比较条件
			// CreateRevision: 键的创建版本号（首次创建时分配，删除后重建会改变）
			// "=", 0: 等于 0 表示键不存在（从未创建或已被删除）
			clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0),
		).
		// Then 分支：如果键不存在（条件为真），执行 Put 操作
		Then(
			// clientv3.OpPut 创建键值对
			clientv3.OpPut(
				lockKey,  // 键名
				"locked", // 键值（可以存储锁的持有者信息，如机器 IP）
				// clientv3.WithLease 将键绑定到租约
				// 绑定后，租约过期时键会自动删除（实现锁的自动释放）
				clientv3.WithLease(leaseResp.ID),
			),
		).
		// Commit 提交事务并执行
		Commit()

	// 检查事务是否执行成功（网络错误、etcd 故障等）
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// ==================== 步骤 4: 判断是否获取锁成功 ====================
	// txnResp.Succeeded 表示 If 条件是否为真
	if txnResp.Succeeded {
		// 条件为真 -> 键不存在 -> Put 操作执行成功 -> 获取锁成功
		t.Logf("Lock acquired")
	} else {
		// 条件为假 -> 键已存在 -> 锁被其他进程持有 -> 需要等待
		t.Logf("Lock is already held, waiting...")

		// ==================== 步骤 5: 监听锁的释放（Watch 机制）====================
		// client.Watch() 监听指定 key 的变化事件
		// 返回一个只读通道，当 key 发生变化时会收到通知
		watchChan := client.Watch(ctx, lockKey)

		// 循环读取监听事件
		for w := range watchChan {
			// w.Events: 事件列表（可能包含多个事件）
			for _, ev := range w.Events {
				// ev.Type: 事件类型
				// clientv3.EventTypeDelete: 键被删除事件
				if ev.Type == clientv3.EventTypeDelete {
					// 锁被释放（键被删除），打印日志
					t.Logf("Lock released, try to acquire again")
					// goto getlock: 跳转到 getlock 标签，重新尝试获取锁
					// 注意：此时租约仍然有效，续约仍在后台运行
					goto getlock
				}
				// 如果是其他事件类型（如 Put），继续等待
			}
		}
		// 如果 Watch 通道关闭（etcd 连接断开、上下文取消），循环退出
	}

	// ==================== 步骤 6: 执行临界区代码 ====================
	// 获取锁成功后，执行业务逻辑
	// Sleep 3 秒模拟耗时操作
	time.Sleep(3 * time.Second)

	// ==================== 步骤 7: 释放锁 ====================
	// 删除锁键，通知其他等待者
	_, err = client.Delete(ctx, lockKey)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}
	// 删除成功后，监听该 key 的其他进程会收到 Delete 事件，触发重试
	t.Logf("Lock released")

	// 函数结束时：
	//   1. defer client.Close() 关闭客户端
	//   2. 租约的续约 goroutine 随着客户端关闭而停止
	//   3. 租约最终过期（如果没有手动撤销）
}
