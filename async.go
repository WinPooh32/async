package async

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

type contextKey string

var (
	contextKeyWG         contextKey = "wg"
	contextKeyCancel     contextKey = "cancel"
	contextKeyErrorsChan contextKey = "errorsCh"
)

var ErrChannelClosed = errors.New("channel is closed")

// Func is a channel writer callback.
type Func[T any] func(chan<- Option[T]) error

// Option is a wrapped pair of value and error.
type Option[T any] struct {
	value T
	err   error
}

// Value unwraps opt's value.
func (opt Option[T]) Value() T { return opt.value }

// Value unwraps opt's error.
func (opt Option[T]) Err() error { return opt.err }

// MakeValue wraps value.
func MakeValue[T any](v T) Option[T] {
	return Option[T]{value: v}
}

// MakeValue wraps error.
func MakeErr[T any](err error) Option[T] {
	return Option[T]{err: err}
}

// Go safely runs function f at a new goroutine. The ch channel will be closed automatically after f returns.
// If panic occurs inside of f it will be recovered and error will be written to the ch channel.
// If capacity is defined or greater than zero, buffered channel will be created.
func Go[T any](ctx context.Context, f Func[T], capacity ...int) <-chan Option[T] {
	var ch chan Option[T]

	if len(capacity) > 0 {
		ch = make(chan Option[T], capacity[0])
	} else {
		ch = make(chan Option[T])
	}

	wg, _ := ctx.Value(contextKeyWG).(*sync.WaitGroup)
	if wg != nil {
		wg.Add(1)
	}

	cancel, _ := ctx.Value(contextKeyWG).(context.CancelFunc)

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		defer close(ch)

		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("recovered panic: %s:\n%s", r, string(debug.Stack()))

				sendErr := sendFailError(ctx, ch, err)
				if sendErr != nil {
					slog.ErrorContext(ctx, "async: failed to send error", slog.String("error", sendErr.Error()))
				}

				if cancel != nil {
					cancel()
				}

				return
			}
		}()

		err := f(ch)
		if err != nil {
			sendErr := sendFailError(ctx, ch, err)
			if sendErr != nil {
				slog.ErrorContext(ctx, "async: failed to send error", slog.String("error", sendErr.Error()))
			}

			if cancel != nil {
				cancel()
			}

			return
		}
	}()

	return ch
}

// Group runs g(i) functions in parallel, their output falls into one channel.
// n is a count of passed functions. i is ranged from 0 to n-1.
func Group[T any](ctx context.Context, g func(i int) Func[T], n int, capacity ...int) <-chan Option[T] {
	fn := func(outCh chan<- Option[T]) error {
		var wg sync.WaitGroup

		wg.Add(n)

		for i := 0; i < n; i++ {
			inCh := Go(ctx, g(i), 1)

			go func() {
				defer wg.Done()

				for v := range inCh {
					outCh <- v
				}
			}()
		}

		wg.Wait()

		return nil
	}

	return Go(ctx, fn, capacity...)
}

// Await reads channel ch and unwraps option to value and error.
// Can be interrupted by closed context.
func Await[T any](ctx context.Context, ch <-chan Option[T]) (value T, err error) {
	wg, _ := ctx.Value(contextKeyWG).(*sync.WaitGroup)
	if wg != nil {
		defer wg.Wait()
	}

	errCh, _ := ctx.Value(contextKeyErrorsChan).(chan error)
	if errCh == nil {
		errCh = make(chan error)
	}

	select {
	case <-ctx.Done():
		return value, ctx.Err()

	case err = <-errCh:
		return value, err

	case opt, ok := <-ch:
		if !ok {
			return value, ErrChannelClosed
		}

		return opt.Value(), opt.Err()
	}
}

// TrySend sends value to the ch channel, blocked until context closed or value passed to the channel.
func TrySend[T any](ctx context.Context, ch chan<- Option[T], value T) (err error) {
	select {
	case ch <- MakeValue(value):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TrySendError sends err to the ch channel, blocked until context closed or value passed to the channel.
func TrySendError[T any](ctx context.Context, ch chan<- Option[T], err error) (_ error) {
	select {
	case ch <- MakeErr[T](err):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sendFailError[T any](ctx context.Context, ch chan<- Option[T], err error) (_ error) {
	errCh, _ := ctx.Value(contextKeyErrorsChan).(chan error)
	if errCh == nil {
		select {
		case ch <- MakeErr[T](err):
			return nil
		default:
			return err
		}
	}

	select {
	case errCh <- err:
		return nil
	default:
		return err
	}
}

type OptFunc func(ctx context.Context) context.Context

// With returns the new context containing optional values from opt funcs and context cancel func.
func With(ctx context.Context, opt ...OptFunc) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	ctx = context.WithValue(ctx, contextKeyCancel, cancel)

	ctx = context.WithValue(ctx, contextKeyErrorsChan, make(chan error, 1))

	for _, o := range opt {
		if o != nil {
			ctx = o(ctx)
		}
	}

	return ctx, cancel
}

func Wait() OptFunc {
	fn := func(ctx context.Context) context.Context {
		wg := new(sync.WaitGroup)
		return context.WithValue(ctx, contextKeyWG, wg)
	}
	return fn
}
