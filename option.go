package retry

import (
	"math/rand"
	"time"
)

type Option func(*Config)

// WithTimes 重试次数, 默认为0表示不重试
func WithTimes(retryTimes int) Option {
	return func(c *Config) {
		c.RetryTimes = retryTimes
	}
}

// WithOnRetryFunc 仅在重试时执行, n代表开始第n次重试
func WithOnRetryFunc(fn OnRetryFunc) Option {
	return func(c *Config) {
		c.OnRetry = fn
	}
}

// WithOnFailedFunc n代表第n次重试(0表示首次调用), err代表第n次重试产生的错误
func WithOnFailedFunc(fn OnFailedFunc) Option {
	return func(c *Config) {
		c.OnFailed = fn
	}
}

// WithDelayStrategy 设置下次重试时间间隔计算函数, 在报错时执行, n代表重试次数(0表示首次调用), err代表重试时产生的错误
func WithDelayStrategy(delayType DelayStrategy) Option {
	return func(c *Config) {
		c.DelayStrategy = delayType
	}
}

// FixedDelay 固定时间间隔
func FixedDelay(delay time.Duration) DelayStrategy {
	return func(n int, err error) time.Duration {
		return delay
	}
}

// LinearDelay 线性时间间隔
func LinearDelay(baseDelay, maxDelay time.Duration) DelayStrategy {
	return func(n int, err error) time.Duration {
		delay := baseDelay * time.Duration(n+1)
		if delay > maxDelay || delay < 0 {
			delay = maxDelay
		}
		return delay
	}
}

// ExponentialDelay 指数时间间隔
func ExponentialDelay(baseDelay, maxDelay time.Duration) DelayStrategy {
	return func(n int, err error) time.Duration {
		delay := baseDelay << n
		if delay > maxDelay || delay < 0 {
			delay = maxDelay
		}
		return delay
	}
}

// RandomDelay 随机时间间隔
func RandomDelay(minDelay, maxDelay time.Duration) DelayStrategy {
	if minDelay < 0 {
		minDelay = 0
	}
	if maxDelay < minDelay {
		maxDelay = minDelay
	}
	return func(n int, err error) time.Duration {
		delay := minDelay
		if maxDelay > minDelay {
			delay += time.Duration(rand.Int63n(int64(maxDelay - minDelay + 1)))
		}
		return delay
	}
}
