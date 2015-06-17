package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
)

const bufsize = 128
const concurrency = 16

func setup_runtime() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

type Buffer struct {
	buf    []byte
	offset int
	cnt    int
}

type ResultType int

const (
	_ = iota
	newline
	match
	bufend
)

type Result struct {
	start int
	value string
	end   int
	rtype ResultType
}

func (r *Result) String() string {
	return fmt.Sprintf("{%d, %s, %d, %d}", r.start, r.value, r.end, r.rtype)
}

func NewBuffer(size int) *Buffer {
	/* This is called a composite literal */
	return &Buffer{make([]byte, size), 0, 0}
}

func (b *Buffer) String() string {
	return string(b.buf[:b.cnt])
}

/* Search a buffer for the string. */
/* These guys should produce little data snippets:
   type Line struct {
	   var start int
	   var value string
	   var end int
   }

   and then the receiver can read a list of these to reconstruct
   line number offsets and print them in order.

   for example:
   {58, "", 63},
   {64, "here it is", 75},
   {76, "", 80}

   would mean: expect 3 snippets, only the second one matched, use these
   offset to calculate line numbers.

   and then we need a gatherer to map line numbers and match them up.
*/
func search(c <-chan *Buffer, resp chan<- *Result, needle string,
	wg *sync.WaitGroup) {
	for {
		b := <-c
		if b == nil {
			wg.Done()
			return
		}
		/* Start the scan */
		offset := b.offset
		var i int
		for i, v := range b.buf {
			if v == '\n' {
				resp <- &Result{offset, "", i, newline}
				offset = i
			}
		}
		resp <- &Result{offset, "", i, bufend}
		fmt.Println(b)
	}
}

func reduce(c <-chan *Result) {
	for {
		r := <-c
		_ = r
	}
}

func getBuffer(f *os.File, offset int) (*Buffer, error) {
	var err error
	buf := NewBuffer(bufsize)
	buf.cnt, err = f.Read(buf.buf)
	buf.offset = offset
	return buf, err
}

func syntax() {
	fmt.Println("pgrep <needle> (haystack is stdin)")
}

func main() {
	var err error
	var buf *Buffer
	var wg sync.WaitGroup
	var needle string

	setup_runtime()

	offset := 0

	if len(os.Args) > 1 {
		needle = os.Args[1]
	} else {
		syntax()
		os.Exit(0)
	}

	buffers := make(chan *Buffer)
	/* This could maybe be buffered at 1 per searcher? */
	resps := make(chan *Result)

	/* Start all the go routines */
	go reduce(resps)
	for i := 0; i < concurrency; i++ {
		go search(buffers, resps, needle, &wg)
	}
	wg.Add(concurrency)

	buf, err = getBuffer(os.Stdin, offset)
	for err == nil {
		offset += buf.cnt
		buffers <- buf
		buf, err = getBuffer(os.Stdin, offset)
	}

	/* Close down all of the go routines */
	for i := 0; i < concurrency; i++ {
		buffers <- nil
	}
	wg.Wait()
}
