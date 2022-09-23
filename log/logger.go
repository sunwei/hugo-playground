package log

import "fmt"

func Process(category string, msg string) {
	fmt.Println("==> Process " + category + ": " + msg)
}
