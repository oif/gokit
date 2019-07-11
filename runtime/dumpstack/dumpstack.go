// dumpstack is a tweak copy from docker
package dumpstack

import (
	"fmt"
	"runtime"
)

func PrintStack() {
	buf := make([]byte, 16384)
	buf = buf[:runtime.Stack(buf, true)]
	fmt.Printf("=== BEGIN goroutine stack dump ===\n%s\n=== END goroutine stack dump ===", buf)
}
