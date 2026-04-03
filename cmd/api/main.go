package main

import (
	"log"
	"net/http"

	"DND-AI-BOT/internal/app"
)

func main() {
	application := app.NewApp()

	if err := http.ListenAndServe(":8080", application.Handler); err != nil {
		log.Fatal(err)
	}
}
