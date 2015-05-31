package main

import (
	"fmt"
	"os"
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
func search(c <-chan *Buffer) {
	for {
		b := <-c
		fmt.Println(b)
	}
}

func getBuffer(f *os.File) (*Buffer, error) {
	var err error
	buf := NewBuffer(bufsize)
	buf.cnt, err = f.Read(buf.buf)
	return buf, err
}

func main() {
	var err error
	var buf *Buffer

	c := make(chan *Buffer)

	/* Start all the go routines */
	for i := 0; i < concurrency; i++ {
		go search(c)
	}

	buf, err = getBuffer(os.Stdin)
	for err == nil {
		c <- buf
		buf, err = getBuffer(os.Stdin)
	}
	fmt.Printf("err is %v\n", err)
}
