package utils

import "log"

// SafeGo runs fn in a goroutine with panic recovery, preventing a panic
// in any background task from crashing the entire process.
func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in background goroutine: %v", r)
			}
		}()
		fn()
	}()
}
