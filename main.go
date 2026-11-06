package main

import (
	"log"

	"mobi2zip/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
