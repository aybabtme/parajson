# parajson

Decode streams of `\n` separated objects in parallel.

# Performance

The gain in speed from using a normal, single threaded decoding is:

| Technique | Throughput  |
|-----------|-------------|
| normal    | 27.30 MB/s  |
| parajson  | 115.07 MB/s |

Using a 386MB file of 1'302'811 `s3.Key` in JSON, on my 8 cores
MBPr 2012.

# Usage

This package implements a simple pipeline to decode JSON objects in
parallel.  Use it like this:


```go
r := getReader()
n := runtime.NumCPU()
protofact := func() interface{} {
    return &s3.Key{}
}

defer r.Close()

keys, errc := parajson.Decode(r, n, protofact)

for proto := range keys {
    key := proto.(*s3.Key)
    // use key
}

for err := range errc {
    log.Fatal(err)
}
```

If you want to use your own decoder (instead of `encoding/json`), pass
your own decoding func:

```go
parajson.SetUnmarshal(func(data []byte, v interface{}) error {
    return ownDecoder(data, v)
})
// do the decoding
```


# License

MIT.
