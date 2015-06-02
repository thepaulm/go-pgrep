package main

import (
	"fmt"
	"os"
	"sync"
)

const bufsize = 128
const concurrency = 16

type Buffer struct {
	buf []byte
	cnt int
}

func NewBuffer(size int) *Buffer {
	/* This is called a composite literal */
	return &Buffer{make([]byte, size), 0}
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
   3,
   {58, "", 63},
   {64, "here it is", 75},
   {76, "", 80}

   would mean: expect 3 snippets, only the second one matched, use these
   offset to calculate line numbers.

   and then we need a gatherer to map line numbers and match them up.
*/
func search(c <-chan *Buffer, needle string, wg *sync.WaitGroup) {
	for {
		b := <-c
		if b == nil {
			wg.Done()
			return
		}
		fmt.Println(b)
	}
}

func getBuffer(f *os.File) (*Buffer, error) {
	var err error
	buf := NewBuffer(bufsize)
	buf.cnt, err = f.Read(buf.buf)
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

	if len(os.Args) > 1 {
		needle = os.Args[1]
	} else {
		syntax()
		os.Exit(0)
	}

	c := make(chan *Buffer)

	/* Start all the go routines */
	for i := 0; i < concurrency; i++ {
		go search(c, needle, &wg)
	}
	wg.Add(concurrency)

	buf, err = getBuffer(os.Stdin)
	for err == nil {
		c <- buf
		buf, err = getBuffer(os.Stdin)
	}

	/* Close down all of the go routines */
	for i := 0; i < concurrency; i++ {
		c <- nil
	}
	wg.Wait()
}
