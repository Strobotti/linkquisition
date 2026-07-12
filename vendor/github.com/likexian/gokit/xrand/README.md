# GoKit - xrand

Rand kits for Golang development.

## Installation

    go get -u github.com/likexian/gokit

## Importing

    import (
        "github.com/likexian/gokit/xrand"
    )

## Documentation

Visit the docs on [GoDoc](https://godoc.org/github.com/likexian/gokit/xrand)

## Example

### Rand int between 0 and 10000

```go
n := xrand.Int(10000)
fmt.Println("rand int between 0 and 10000 is:", n)
```

### Rand int between 1000 and 10000

```go
n := xrand.IntRange(1000, 10000)
fmt.Println("rand int between 1000 and 10000 is:", n)
```

### Rand bytes with length of 10

```go
b, err := xrand.Bytes(10)
if err != nil {
    fmt.Println("rand bytes:", b)
}
```

## License

Copyright 2012-2026 [Li Kexian](https://www.likexian.com/)

Licensed under the Apache License 2.0

## Donation

If this project is helpful, please share it with friends.

If you want to thank me, you can [give me a cup of coffee](https://www.likexian.com/donate/).
