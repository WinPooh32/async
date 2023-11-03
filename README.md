# Async

![test](https://github.com/WinPooh32/async/actions/workflows/test.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/WinPooh32/async.svg)](https://pkg.go.dev/github.com/WinPooh32/async)

Run async code with safe!

## Example

```Go
package main

import (
	"context"
	"fmt"

	"github.com/WinPooh32/async"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fn := func(ch chan<- async.Option[string]) error {
		err := async.TrySend(ctx, ch, "Hello Async!")
		if err != nil {
			return err
		}

		return nil
	}

	ch := async.Go(ctx, fn)

	value, err := async.Await(ctx, ch)
	if err != nil {
		panic(err)
	}

	fmt.Println(value)
}
```

Look for more examples at **async_test.go**
