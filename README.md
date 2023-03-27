# Async

![test](https://github.com/WinPooh32/async/actions/workflows/test.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/WinPooh32/async.svg)](https://pkg.go.dev/github.com/WinPooh32/async)

Run async code with safe!

## Example

```Go
package main

import (
	"fmt"

	"github.com/WinPooh32/async"
)

func main() {
	ch := async.Go(func(ch chan<- async.Option[string]) {
		ch <- async.MakeValue("Hello Async!")
	})

	value, err := async.Await(ch)
	if err != nil {
		panic(err)
	}

	fmt.Println(value)
}
```

Look for more examples at **async_test.go**