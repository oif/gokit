package runtime

import "fmt"

type crashHandler func(interface{})

var AlwaysHandle = []crashHandler{
	crashLog,
}

func HandleCrash(handlers ...crashHandler) {
	if e := recover(); e != nil {
		for _, handle := range AlwaysHandle {
			handle(e)
		}
		for _, handle := range handlers {
			handle(e)
		}
		panic(e)
	}
}

func crashLog(e interface{}) {
	fmt.Printf("[PANIC] %v\n", e)
}
