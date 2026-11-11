package reactive

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
	"sync"

	"github.com/petermattis/goid"
)

func currentGoroutine() int64 { return goid.Get() }

func stackGoroutine() (int64, error) {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		return 0, fmt.Errorf("reactive: unparseable stack header %q", b)
	}
	return strconv.ParseInt(string(b[:i]), 10, 64)
}

func VerifyGoid() error {
	here, err := stackGoroutine()
	if err != nil {
		return err
	}
	if got := currentGoroutine(); got != here {
		return fmt.Errorf("reactive: goid %d does not match runtime.Stack %d", got, here)
	}
	var wg sync.WaitGroup
	var other, otherStack int64
	var otherErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		other = currentGoroutine()
		otherStack, otherErr = stackGoroutine()
	}()
	wg.Wait()
	if otherErr != nil {
		return otherErr
	}
	if other != otherStack || other == here {
		return fmt.Errorf("reactive: goid %d does not match runtime.Stack %d on a second goroutine", other, otherStack)
	}
	return nil
}
