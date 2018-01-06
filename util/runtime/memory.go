package runtime

import (
	"log"
	"runtime"
)

func LogMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("\nAlloc = %v KB\nStackSys = %v KB\nSys = %v KB \nNumGC = %v\n\n", m.Alloc/1024, m.StackSys/1024, m.Sys/1024, m.NumGC)

}
