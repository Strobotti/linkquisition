# GoKit - xslice

Slice kits for Golang development.

## Installation

    go get -u github.com/likexian/gokit

## Importing

    import (
        "github.com/likexian/gokit/xslice"
    )

## Documentation

Visit the docs on [GoDoc](https://godoc.org/github.com/likexian/gokit/xslice)

## Example

### Get unique of string array

```go
array := xslice.Unique([]string{"a", "a", "b", "b", "b", "c"})
fmt.Println("new array:", array)
```

### Get unique of int array

```go
array := xslice.Unique([]int{0, 0, 1, 1, 1, 2, 2, 3})
fmt.Println("new array:", array)
```

## License

Copyright 2012-2026 [Li Kexian](https://www.likexian.com/)

Licensed under the Apache License 2.0

## Donation

If this project is helpful, please share it with friends.

If you want to thank me, you can [give me a cup of coffee](https://www.likexian.com/donate/).
