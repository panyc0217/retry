# Retry

简洁的 Golang 重试库

- **上下文支持**: 完全支持 `context.Context`，可随时取消或超时
- **灵活配置**: 使用 Options 模式，配置简洁直观
- **延迟策略**: 内置固定延迟和指数退避策略，支持自定义策略
- **回调机制**: 提供重试前和失败后的回调函数
- **提前终止**: 支持 `Break` 函数立即中断重试循环
- **零依赖**: 核心代码无外部依赖

## 安装

```bash
go get github.com/panyc2000/retry
```

## 快速开始

### 基本用法

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/panyc2000/retry"
)

func main() {
	err := retry.Do(context.Background(),
		func() error {
			err := do()
			if err != nil {
				// 不可重试异常
				if errors.Is(err, ErrAuthFailed) {
					return retry.Break(err)
				}
				// 可重试异常
				return err
			}
			// 正常结束
			return nil
		},
		retry.WithTimes(3),                                       // 重试3次
		retry.WithDelayStrategy(retry.FixedDelay(1*time.Second)), // 重试间隔固定为1s
		retry.WithOnRetryFunc(func(n int) {
			fmt.Printf("开始第 %d 次重试\n", n)
		}),
		retry.WithOnFailedFunc(func(n int, err error) {
			fmt.Printf("第 %d 次执行失败: %v\n", n, err)
		}),
	)
	if err != nil {
		fmt.Println("重试失败:", err)
	}
}

```


## API 文档

### 配置选项

#### `WithTimes(retryTimes int)`

设置重试次数，默认为 0（不重试），如果设置为 3，则最多执行 4 次（1 次初始执行 + 3 次重试）。

#### `WithOnRetryFunc(fn OnRetryFunc)`

设置重试前的回调函数（参数 `n` 表示即将开始第 n 次重试，n 从 1 开始），仅在重试时执行。

#### `WithOnFailedFunc(fn OnFailedFunc)`

设置执行失败后的回调函数（参数 `n` 表示第 n 次执行，n 从 0 开始；参数 `err` 为该次执行产生的错误）。

#### `WithDelayStrategy(delayType DelayStrategy)`

设置重试延迟策略，用于计算下次重试前的等待时间。

内置重试延迟策略：
1. `FixedDelay(delay time.Duration)`：固定时间间隔
2. `LinearDelay(baseDelay, maxDelay time.Duration)`：线性时间间隔，重试延迟时间呈现线性增长
3. `ExponentialDelay(baseDelay, maxDelay time.Duration)`：指数时间间隔，重试延迟时间以2的指数倍增长
4. `RandomDelay(minDelay, maxDelay time.Duration)`：随机时间间隔

自定义延迟策略：
```go
// 自定义策略：根据错误类型决定延迟时间
customDelay := func (n int, err error) time.Duration {
    if errors.Is(err, ErrRateLimit) {
        return 5 * time.Second
    }
    return time.Second
}
err := retry.Do(context.Background(),
    fn,
    retry.WithTimes(3),
    retry.WithDelayStrategy(customDelay),
)
```

### 核心函数

#### `Do(ctx context.Context, fn func() error, opts ...Option) error`

执行函数 `fn` 并在失败时重试，函数返回最后一次执行返回的错误。可使用 `Break(err error) error` 中断重试循环。
