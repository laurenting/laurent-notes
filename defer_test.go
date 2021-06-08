package service

import (
	"fmt"
	"io"
	"testing"
)

func a() error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("defer1", err)
		} else {
			fmt.Println("ok")
		}
	}()

	err = io.ErrNoProgress

	defer func() {
		err = io.ErrShortWrite
		if err != nil {
			fmt.Println("defer2", err)
		} else {
			fmt.Println("ok")
		}
	}()
	err = io.ErrClosedPipe

	fmt.Println("err", err)
	return err
}
func TestDefer(t *testing.T) {
	fmt.Println(a())
}
