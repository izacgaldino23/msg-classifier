// Package main is the terminal entrypoint (DC-009). It reads a line, classifies it
// through the same core as cmd/web and cmd/api, and prints the outcome as text.
package main

import (
	"fmt"
	"log"
	"os"

	"msg-classifier/internal/app"
	"msg-classifier/internal/cli"
	"msg-classifier/internal/config"
)

func main() {
	application, err := app.New(config.GetEnv().DBPath)
	if err != nil {
		log.Fatalf("failed to build application: %v", err)
	}

	fmt.Println("msg-classifier · digite uma mensagem, ou 'exit' para sair")

	runner := cli.New(application.Classifier.Classify, application.Dispatcher.Dispatch, os.Stdin, os.Stdout)
	if err := runner.Run(); err != nil {
		log.Fatalf("cli stopped: %v", err)
	}
}