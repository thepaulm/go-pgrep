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
/*
	These guys should produce little Result structs
	and then the receiver can read a list of these to reconstruct
	line number offsets and print them in order.

	for example:
	{58, "", 63, newline},
	{63, "", 68, bufend},
	{68, "here it is", 78, match},
	{78, "", 80, bufend}

	would mean:
	58 to 63: no match, newline at 63
	63 to 68: no match, no newline
	68 to 78: contains the string
	78 to 80: no match, no newline

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
	endm := make(map[int]*Result)
	startm := make(map[int]*Result)
	lineno := 0
	lowchar := 0
	_ = lineno
	_ = lowchar
	for {
		r := <-c
		/* If we have something in the end map that ends where we started ... */
		exist, ok := endm[r.start]
		if ok == true {
			exist.value += r.value  // Append the str
			exist.end = r.end       // Add up the new end
			endm[exist.end] = exist // We now have a new end
			delete(endm, r.start)   // Remove the old end
		} else {
			/* We don't, so just add a new start entry */
			startm[r.start] = r
		}
		/* If we have something in the start map that begins where we ended ... */
		exist, ok = startm[r.end]
		if ok == true {
			r.value += exist.value      // Append the str
			r.end = exist.end           // Add up the new end
			startm[r.start] = r         // Our new thing has a start
			delete(startm, exist.start) // We've prepended, so the old is gone
			endm[r.end] = r             // Update the end map for this end
		} else {
			/* We don't, so just add a new end entry */
			endm[r.end] = r
		}
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
