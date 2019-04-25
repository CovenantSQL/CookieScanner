Hash Stable Pack
=======
This is a code generation tool for **QUICK** struct content compare or hash computation. 

### For
- Quick compare nested struct without reflection (10~20 times faster)

    ```go
    BenchmarkCompare/benchmark_reflect-8         	  100000	      20074 ns/op //reflect.DeepEqual
    BenchmarkCompare/benchmark_hsp-8             	  500000	       2322 ns/op
    BenchmarkCompare/benchmark_hsp_1_cached-8    	 1000000	       1101 ns/op
    BenchmarkCompare/benchmark_hsp_both_cached-8   100000000	       11.2 ns/op
    ```
    bench cases see [here](test/hashstable_test.go)
    
- Quick calculation of struct hash or signature without reflection. used in [CovenantSQL](https://github.com/CovenantSQL/CovenantSQL) for block hash.

### How

Basically it will generate an `MarshalHash` method which follow the [MessagePack Spec](https://github.com/msgpack/msgpack/blob/master/spec.md) but :

1. Without the struct key.
1. Stable output of map.
1. Can be used to compare different type with same hsp tag.


That is the following 2 structs with different member name

For more: see [test cases](test)
```go
//go:generate hsp

type Person1 struct {
	Name       string
	Age        int
	Address    string
	Map        map[string]int
	unexported bool             // this field is ignored
	Unexported string `hsp:"-"` // this field is ignored
}

type Person2 struct {
	Name       string
	Address    string
	Age        int
	Map222     map[string]int `hspack:"Map"`
	unexported bool             // this field is ignored
	Unexported string `hsp:"-"` // this field is ignored
}
```

But with the same name and content of exported member, `MarshalHash` will produce the same bytes array:
```go
package person

import (
	"bytes"
	"testing"
)

func TestMarshalHashAccountStable3(t *testing.T) {
	p1 := Person1{
		Name:       "Auxten",
		Address:    "@CovenantSQL.io",
		Age:        70,
		Map:         map[string]int{"ss": 2, "s": 1, "sss": 3},
		unexported: false,
	}
	p2 := Person2{
		Name:       "Auxten",
		Address:    "@CovenantSQL.io",
		Age:        70,
		Map222:      map[string]int{"ss": 2, "s": 1, "sss": 3},
		unexported: true,
	}
	bts1, err := p1.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	bts2, err := p2.MarshalHash()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bts1, bts2) {
		t.Fatal("hash not stable")
	}
}
```
the order of struct member is sorted by struct tag (if not, use name) 


You can read more about MessagePack [in the wiki](http://github.com/tinylib/msgp/wiki), or at [msgpack.org](http://msgpack.org).

### Why?

- Use Go as your schema language
- Performance

### Why not?

- MessagePack: member name is unnecessary, different implementation may add some fields which made result undetermined. And also golang's map...
- Prorobuf: struct must defined in proto language, and other limitations discussed [here](https://gist.github.com/kchristidis/39c8b310fd9da43d515c4394c3cd9510)

### Quickstart

1. Quick Install
```bash
go get -u github.com/CovenantSQL/HashStablePack/hsp
```

2. Add tag for source
In a source file, include the following directive:

```go
//go:generate hsp
```

3. Run go generate
```bash
go generate ./...
```

The `hsp` command will generate serialization methods for all exported type declarations in the file.

By default, the code generator will only generate `MarshalHash` and `Msgsize` method
```go
func (z *Test) MarshalHash() (o []byte, err error)
func (z *Test) Msgsize() (s int)
```


### Features

 - Extremely fast generated code
 - Test and benchmark generation
 - Support for complex type declarations
 - Native support for Go's `time.Time`, `complex64`, and `complex128` types 
 - Support for arbitrary type system extensions
 - File-based dependency model means fast codegen regardless of source tree size.


### License

This lib is inspired by https://github.com/tinylib/msgp
Most Code is diverted from https://github.com/tinylib/msgp, but It's an total different lib for usage. So I created a new project instead of forking it.

