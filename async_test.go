package async_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/WinPooh32/async"
)

func TestGo_Value(t *testing.T) {
	const testValue = 1

	ch := async.Go(func(ch chan<- async.Option[int]) {
		ch <- async.MakeValue(testValue)
	})

	opt := <-ch
	err := opt.Err()
	if err != nil {
		t.Error(err)

		return
	}

	value := opt.Value()

	if value != testValue {
		t.Fail()

		return
	}
}

func TestGo_Err(t *testing.T) {
	testErr := errors.New("test error")

	ch := async.Go(func(ch chan<- async.Option[int]) {
		ch <- async.MakeErr[int](testErr)
	})

	opt := <-ch
	err := opt.Err()
	if err != nil {
		if err != testErr {
			t.Fail()
		}

		return
	}

	t.Fail()
}

func TestGo_Panic(t *testing.T) {
	ch := async.Go(func(ch chan<- async.Option[int]) {
		panic("something went wrong!")
	})

	opt := <-ch
	err := opt.Err()
	if err != nil {
		return
	}

	t.Fail()
}

func TestGo_Stream(t *testing.T) {
	const testValue = 1
	const testSum = 10

	ch := async.Go(func(ch chan<- async.Option[int]) {
		for i := 0; i < testSum; i++ {
			ch <- async.MakeValue(testValue)
		}
	})

	var sum int

	for opt := range ch {
		err := opt.Err()
		if err != nil {
			t.Error(err)

			return
		}

		value := opt.Value()

		if value != testValue {
			t.Fail()

			return
		}

		sum += value
	}

	if sum != testSum {
		t.Fail()

		return
	}
}

func TestGo_StreamBuffered(t *testing.T) {
	const testValue = 1
	const testSum = 10
	const testChCapacity = 100

	ch := async.Go(func(ch chan<- async.Option[int]) {
		for i := 0; i < testSum; i++ {
			ch <- async.MakeValue(testValue)
		}
	}, testChCapacity)

	var sum int

	for opt := range ch {
		err := opt.Err()
		if err != nil {
			t.Error(err)

			return
		}

		value := opt.Value()

		if value != testValue {
			t.Fail()

			return
		}

		sum += value
	}

	if sum != testSum {
		t.Fail()

		return
	}
}

func TestGo_Sync(t *testing.T) {
	chA := async.Go(func(ch chan<- async.Option[string]) {
		ch <- async.MakeValue("A")
	})

	chB := async.Go(func(ch chan<- async.Option[string]) {
		ch <- async.MakeValue("B")
	})

	chC := async.Go(func(ch chan<- async.Option[string]) {
		ch <- async.MakeValue("C")
	})

	var errs []error

	optA := <-chA
	if err := optA.Err(); err != nil {
		errs = append(errs, err)
	}

	optB := <-chB
	if err := optB.Err(); err != nil {
		errs = append(errs, err)
	}

	optC := <-chC
	if err := optC.Err(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		t.Fail()
	}

	a := optA.Value()
	b := optB.Value()
	c := optC.Value()

	if a != "A" {
		t.Fail()
	}

	if b != "B" {
		t.Fail()
	}

	if c != "C" {
		t.Fail()
	}
}

func TestAwait(t *testing.T) {
	const testValue = 1

	ch := async.Go(func(ch chan<- async.Option[int]) {
		ch <- async.MakeValue(testValue)
	})

	v, err := async.Await(ch)
	if err != nil {
		t.Error(err)

		return
	}

	if v != 1 {
		t.Fail()

		return
	}
}

func TestAwaitContext_Value(t *testing.T) {
	const testValue = 1

	ch := async.Go(func(ch chan<- async.Option[int]) {
		ch <- async.MakeValue(testValue)
	})

	v, err := async.AwaitContext(context.Background(), ch)
	if err != nil {
		t.Error(err)

		return
	}

	if v != 1 {
		t.Fail()

		return
	}
}

func TestAwaitContext_CanceledContext(t *testing.T) {
	const testValue = 1

	ch := async.Go(func(ch chan<- async.Option[int]) {
		<-time.After(10 * time.Second)
		ch <- async.MakeValue(testValue)
	})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-time.After(50 * time.Millisecond)
		cancel()
	}()

	_, err := async.AwaitContext(ctx, ch)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		t.Error(err)
	}

	t.Fail()
}

func TestAwaitContext_Err(t *testing.T) {
	ch := async.Go(func(ch chan<- async.Option[int]) {
		// Close channel without value at return.
	})

	_, err := async.AwaitContext(context.Background(), ch)
	if err != nil {
		if errors.Is(err, async.ErrChannelClosed) {
			return
		}
		t.Error(err)
	}

	t.Fail()
}
