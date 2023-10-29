package async

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
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
func Go[T any](f Func[T], capacity ...int) <-chan Option[T] {
	var ch chan Option[T]

	if len(capacity) > 0 {
		ch = make(chan Option[T], capacity[0])
	} else {
		ch = make(chan Option[T])
	}

	go func() {
		defer close(ch)

		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("recovered panic: %s:\n%s", r, string(debug.Stack()))
				ch <- MakeErr[T](err)

				return
			}
		}()

		err := f(ch)
		if err != nil {
			ch <- MakeErr[T](err)

			return
		}
	}()

	return ch
}

// Group runs g(i) functions in parallel, their output falls into one channel.
// n is a count of passed functions. i is ranged from 0 to n-1.
func Group[T any](g func(i int) Func[T], n int, capacity ...int) <-chan Option[T] {
	fn := func(outCh chan<- Option[T]) error {
		var wg sync.WaitGroup

		wg.Add(n)

		for i := 0; i < n; i++ {
			inCh := Go(g(i), 1)

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

	return Go(fn, capacity...)
}

// Await reads channel ch and unwraps option to value and error.
func Await[T any](ch <-chan Option[T]) (value T, err error) {
	return AwaitContext(context.Background(), ch)
}

// AwaitContext reads channel ch and unwraps option to value and error.
// Can be interrupted by closed context.
func AwaitContext[T any](ctx context.Context, ch <-chan Option[T]) (value T, err error) {
	select {
	case <-ctx.Done():
		return value, ctx.Err()

	case opt, ok := <-ch:
		if !ok {
			return value, ErrChannelClosed
		}

		return opt.Value(), opt.Err()
	}
}
