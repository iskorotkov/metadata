# Metadata

Marshal/unmarshal Kubernetes `ObjectMeta` properties into/from a custom `struct`.

## Features

Supported `ObjectMeta` properties:

- labels
- annotations

Supported types:

- `string`
- `bool`
- `time.Duration`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`
- `complex64`, `complex128`
- `slice` of:
    - `string`
    - `bool`
    - `int`
    - `uint`
    - `float32`
    - `float64`
    - `time.Duration`

## Examples

```go
package main

import (
	"github.com/iskorotkov/metadata"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
)

// Add tags to struct fields
type Data struct {
	ID     int      `annotation:"id"`
	Name   string   `annotation:"name"`
	Age    uint     `label:"age"`
	Skills []string `label:"skills"`
}

func main() {
	// Create a struct with metadata
	d := Data{
		ID:     1,
		Name:   "John",
		Age:    20,
		Skills: []string{"reading", "driving", "programming"},
	}

	// Use prefix to distinguish your labels and annotations from the ones from other tools or Kubernetes
	// Keys are formatted like this:
	// {prefix}/{name}
	prefix := "my-org.com"

	// Marshal metadata struct: add labels and annotations to ObjectMeta
	meta := v1.ObjectMeta{}
	if err := metadata.Marshal(&meta, &d, prefix); err != nil {
		panic(err)
	}

	// Unmarshall metadata: read labels and annotations and assign their values to struct fields
	d2 := Data{}
	if err := metadata.Unmarshal(meta, &d2, prefix); err != nil {
		panic(err)
	}

	// d and d2 are equal
	if !reflect.DeepEqual(d, d2) {
		panic("something doesn't work")
	}
}
```
