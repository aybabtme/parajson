package parajson_test

import (
	"encoding/json"
	"fmt"
	"github.com/aybabtme/goamz/s3"
	"github.com/aybabtme/parajson"
	"github.com/dustin/go-humanize"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"testing"
	"time"
)

func BenchmarkParajson(b *testing.B) {
	runtime.GC()
	log.SetOutput(ioutil.Discard)
	n := runtime.NumCPU()
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(n))

	r, readersize := getReader()
	defer r.Close()

	b.ResetTimer()
	keys, errc := parajson.Decode(r, n, func() interface{} {
		return &s3.Key{}
	})

	_, _ = printStats(keys, errc, readersize)

	b.SetBytes(int64(readersize))
}

func BenchmarkNormal(b *testing.B) {
	runtime.GC()
	log.SetOutput(ioutil.Discard)
	n := runtime.NumCPU()
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(n))

	r, readersize := getReader()
	defer r.Close()

	start := time.Now()
	var key s3.Key
	var keys []s3.Key

	b.ResetTimer()
	dec := json.NewDecoder(r)
decoding:
	for {
		err := dec.Decode(&key)
		switch err {
		case io.EOF:
			break decoding
		case nil:
			keys = append(keys, key)
		default:
			b.Fatalf("error decoding: %v", err)
		}
	}

	_, _ = printStatsSlice(keys, readersize, start)

	b.SetBytes(int64(readersize))

}

func ExampleDecode_S3keys() {

	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	r, readersize := getReader()
	defer r.Close()

	keys, errc := parajson.Decode(r, n, func() interface{} {
		return &s3.Key{}
	})

	count, keysize := printStats(keys, errc, readersize)

	fmt.Printf("%d s3 keys, storing %s", count, humanize.Bytes(keysize))

	// Output:
	// 1302811 s3 keys, storing 107GB
}

func getReader() (*os.File, uint64) {
	file, err := os.Open("s3_key_list.json")
	if err != nil {
		log.Fatalf("opening file: %v", err)
	}

	fi, err := file.Stat()
	if err != nil {
		log.Fatalf("stating file: %v", err)
	}
	return file, uint64(fi.Size())
}

func printStatsSlice(keys []s3.Key, filesize uint64, start time.Time) (count, keysize uint64) {

	for _, key := range keys {
		count++
		keysize += uint64(key.Size)
	}

	done := time.Since(start)
	log.Printf("done in %v", done)

	persec := uint64(float64(filesize) / done.Seconds())
	log.Printf("%s keys of %s at %s/s",
		humanize.Comma(int64(count)),
		humanize.Bytes(filesize),
		humanize.Bytes(persec))

	return count, keysize
}

func printStats(keys <-chan interface{}, errc <-chan error, filesize uint64) (count, keysize uint64) {
	start := time.Now()

	for keyproto := range keys {
		key := keyproto.(*s3.Key)
		count++
		keysize += uint64(key.Size)
	}

	for err := range errc {
		fmt.Printf("decoding error: %v\n", err)
	}
	done := time.Since(start)
	log.Printf("done in %v", done)

	persec := uint64(float64(filesize) / done.Seconds())
	log.Printf("%s keys of %s at %s/s",
		humanize.Comma(int64(count)),
		humanize.Bytes(filesize),
		humanize.Bytes(persec))

	return count, keysize
}
