package retry

import (
	"context"
	"time"
)

// OnRetryFunc 重试前回调, 第n次执行前调用(n=0时不调用)
type OnRetryFunc func(n int)

// OnFailedFunc 执行失败回调, 第n次执行失败后调用(n=0时会调用)
type OnFailedFunc func(n int, err error)

// DelayStrategy 重试间隔策略, 第n次执行失败后调用(n=0时会调用)
type DelayStrategy func(n int, err error) time.Duration

type Config struct {
	RetryTimes    int
	OnRetry       OnRetryFunc
	OnFailed      OnFailedFunc
	DelayStrategy DelayStrategy
}

func NewConfig(opts ...Option) *Config {
	config := Config{}
	for _, opt := range opts {
		opt(&config)
	}
	return &config
}

type breakError struct {
	error
}

func Break(err error) error {
	return breakError{err}
}

func (config *Config) Do(ctx context.Context, fn func() error) error {

	if err := ctx.Err(); err != nil {
		return err
	}

	if config.OnRetry == nil {
		config.OnRetry = func(n int) {}
	}

	if config.OnFailed == nil {
		config.OnFailed = func(n int, err error) {}
	}

	if config.DelayStrategy == nil {
		config.DelayStrategy = FixedDelay(0)
	}

	var n int
	for {
		if n > 0 {
			config.OnRetry(n)
		}

		err := fn()

		v, breakRetry := err.(breakError)
		if breakRetry {
			err = v.error
		}

		if err == nil {
			return nil
		}

		config.OnFailed(n, err)

		if n >= config.RetryTimes {
			breakRetry = true
		}

		if breakRetry {
			return err
		}

		select {
		case <-time.After(config.DelayStrategy(n, err)):
			n++
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func Do(ctx context.Context, fn func() error, opts ...Option) error {
	return NewConfig(opts...).Do(ctx, fn)
}
