// Package parajson implements a parallel decoder of JSON objects.
// It expects to read a unique JSON object per `\n` separated line in
// the io.Reader it consumes.
//
// For better performance, set `runtime.GOMAXPROCS` to be `n`.
package parajson

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"sync"
)

var (
	lock sync.RWMutex
	elog = log.New(ioutil.Discard, "[parajson] ", 0)

	unmarshaller = json.Unmarshal
)

// LogOut sets the internal logger to use the given output.
func LogOut(out io.Writer) {
	lock.Lock()
	elog = log.New(out, "[parajson] ", 0)
	lock.Unlock()
}

// SetUnmarshal to use your own unmarshaling func. For instance, to use
// a fatherhood decoder.
func SetUnmarshal(u func(line []byte, v interface{}) error) {
	lock.Lock()
	unmarshaller = u
	lock.Unlock()
}

// Decode the JSON objects in r using n goroutines, unmarshaling the content
// line by line into prototypes given by protofactory.
//
// The prototypes of protofactory should not be used by the client. Decode
// uses them to avoid decoding into `map[string]interface{}`.
func Decode(r io.Reader, n int, protofactory func() interface{}) (<-chan interface{}, <-chan error) {
	lock.RLock()
	defer lock.RUnlock()

	lines := make(chan []byte, n*10)
	out := make(chan interface{}, n*10)
	errc := make(chan error, n+1)

	go func() {

		wg := sync.WaitGroup{}
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(w *sync.WaitGroup, id int) {
				defer w.Done()
				err := decode(lines, out, protofactory)
				if err != nil {
					elog.Printf("decoder %d: %v", id, err)
					errc <- err
					return
				}
			}(&wg, i)
		}

		err := readLines(r, lines)
		if err != nil {
			elog.Printf("readlines: %v", err)
			errc <- err
		}
		close(lines)

		wg.Wait()
		close(out)
		close(errc)
	}()

	return out, errc
}

func decode(lines <-chan []byte, out chan<- interface{}, protofactory func() interface{}) error {
	for line := range lines {
		proto := protofactory()
		err := unmarshaller(line, proto)
		if err != nil {
			return err
		}
		out <- proto
	}
	return nil
}

func readLines(r io.Reader, lines chan<- []byte) error {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadBytes('\n')
		switch err {
		case io.EOF:
			return nil
		case nil:
			lines <- line
		default:
			// not EOF, not nil
			return err
		}
	}
}
