package main

import "log"

func main() {
	if err := run(); err != nil {
		log.Fatalf("[BOOT] action=run event=failed error=%v", err)
	}
}
