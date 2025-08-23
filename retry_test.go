package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testErr = errors.New("test")

func SuccessOnMaxCallFunc(maxCall int) func() error {
	call := 0
	return func() error {
		call++
		if call >= maxCall {
			return nil
		}
		return testErr
	}
}

func TestDo(t *testing.T) {
	type args struct {
		ctxFunc func() (context.Context, context.CancelFunc)
		fn      func() error
		opts    []Option
	}
	type expected struct {
		returnErr     error
		onRetryCount  int
		onFailedCount int
		duration      time.Duration
	}
	var onRetryCount int
	var onFailedCount int
	onRetryFunc := func(n int) {
		onRetryCount++
		assert.Equal(t, onRetryCount, n)
	}
	onFailedFunc := func(n int, err error) {
		assert.Equal(t, onFailedCount, n)
		onFailedCount++
	}
	for _, testCase := range []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "default",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      func() error { return testErr },
				opts:    []Option{},
			},
			expected: expected{
				returnErr:     testErr,
				onRetryCount:  0,
				onFailedCount: 0,
				duration:      0,
			},
		},
		{
			name: "all failed",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      func() error { return testErr },
				opts: []Option{
					WithTimes(10),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
				},
			},
			expected: expected{
				returnErr:     testErr,
				onRetryCount:  10,
				onFailedCount: 11,
				duration:      0,
			},
		},
		{
			name: "fixed delay 1",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      SuccessOnMaxCallFunc(2),
				opts: []Option{
					WithTimes(3),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(FixedDelay(time.Second)),
				},
			},
			expected: expected{
				returnErr:     nil,
				onRetryCount:  1,
				onFailedCount: 1,
				duration:      time.Second,
			},
		},
		{
			name: "fixed delay 2",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      SuccessOnMaxCallFunc(5),
				opts: []Option{
					WithTimes(3),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(FixedDelay(time.Second)),
				},
			},
			expected: expected{
				returnErr:     testErr,
				onRetryCount:  3,
				onFailedCount: 3,
				duration:      3 * time.Second,
			},
		},
		{
			name: "backoff delay 1",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      SuccessOnMaxCallFunc(4),
				opts: []Option{
					WithTimes(5),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(ExponentialDelay(time.Second, 3*time.Second)),
				},
			},
			expected: expected{
				returnErr:     nil,
				onRetryCount:  3,
				onFailedCount: 3,
				duration:      time.Second + 2*time.Second + 3*time.Second,
			},
		},
		{
			name: "linear delay 1",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      SuccessOnMaxCallFunc(4),
				opts: []Option{
					WithTimes(5),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(LinearDelay(time.Second, 10*time.Second)),
				},
			},
			expected: expected{
				returnErr:     nil,
				onRetryCount:  3,
				onFailedCount: 3,
				// n=0: 1s, n=1: 2s, n=2: 3s => total 6s
				duration: time.Second + 2*time.Second + 3*time.Second,
			},
		},
		{
			name: "linear delay with max",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) { return context.Background(), func() {} },
				fn:      SuccessOnMaxCallFunc(4),
				opts: []Option{
					WithTimes(5),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(LinearDelay(time.Second, 2*time.Second)),
				},
			},
			expected: expected{
				returnErr:     nil,
				onRetryCount:  3,
				onFailedCount: 3,
				// n=0: 1s, n=1: 2s, n=2: 2s (capped) => total 5s
				duration: time.Second + 2*time.Second + 2*time.Second,
			},
		},
		{
			name: "canceled context",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx, cancel
				},
				fn: SuccessOnMaxCallFunc(0),
				opts: []Option{
					WithTimes(5),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(ExponentialDelay(time.Second, 5*time.Second)),
				}},
			expected: expected{
				returnErr:     context.Canceled,
				onRetryCount:  0,
				onFailedCount: 0,
				duration:      0,
			},
		},
		{
			name: "timeout context",
			args: args{
				ctxFunc: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(context.Background(), 5*time.Second)
				},
				fn: SuccessOnMaxCallFunc(4),
				opts: []Option{
					WithTimes(5),
					WithOnRetryFunc(onRetryFunc),
					WithOnFailedFunc(onFailedFunc),
					WithDelayStrategy(ExponentialDelay(time.Second, 5*time.Second)),
				}},
			expected: expected{
				returnErr:     context.DeadlineExceeded,
				onRetryCount:  2,
				onFailedCount: 4,
				duration:      5 * time.Second,
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			onRetryCount = 0
			onFailedCount = 0
			s := time.Now()
			ctx, cancel := testCase.args.ctxFunc()
			defer cancel()
			err := Do(ctx, testCase.args.fn, testCase.args.opts...)
			e := time.Now()
			duration := e.Sub(s)
			assert.Equal(t, testCase.expected.returnErr, err)
			assert.Greater(t, duration, testCase.expected.duration-100*time.Millisecond)
			assert.Less(t, duration, testCase.expected.duration+100*time.Millisecond)
		})
	}
}

func TestRandomDelay(t *testing.T) {
	t.Run("random delay within range", func(t *testing.T) {
		minDelay := 100 * time.Millisecond
		maxDelay := 200 * time.Millisecond
		exec := 0
		s := time.Now()
		err := Do(context.Background(), func() error {
			exec++
			if exec >= 4 {
				return nil
			}
			return testErr
		}, WithTimes(5), WithDelayStrategy(RandomDelay(minDelay, maxDelay)))
		duration := time.Since(s)
		assert.Nil(t, err)
		assert.Equal(t, 4, exec)
		// 3 delays, each between 100-200ms => total 300-600ms
		assert.GreaterOrEqual(t, duration, 3*minDelay-50*time.Millisecond)
		assert.LessOrEqual(t, duration, 3*maxDelay+50*time.Millisecond)
	})

	t.Run("random delay with same min max", func(t *testing.T) {
		delay := 100 * time.Millisecond
		exec := 0
		s := time.Now()
		err := Do(context.Background(), func() error {
			exec++
			if exec >= 3 {
				return nil
			}
			return testErr
		}, WithTimes(5), WithDelayStrategy(RandomDelay(delay, delay)))
		duration := time.Since(s)
		assert.Nil(t, err)
		assert.Equal(t, 3, exec)
		// 2 delays, each exactly 100ms => total 200ms
		assert.Greater(t, duration, 2*delay-50*time.Millisecond)
		assert.Less(t, duration, 2*delay+50*time.Millisecond)
	})
}

func TestBreak(t *testing.T) {
	t.Run("break with error", func(t *testing.T) {
		exec := 0
		err := Do(context.Background(), func() error {
			exec++
			return Break(testErr)
		}, WithTimes(10))
		assert.Equal(t, testErr, err)
		assert.Equal(t, 1, exec)
	})

	t.Run("break with nil", func(t *testing.T) {
		exec := 0
		err := Do(context.Background(), func() error {
			exec++
			return Break(nil)
		},
			WithTimes(10),
		)
		assert.Nil(t, err)
		assert.Equal(t, 1, exec)
	})

}
