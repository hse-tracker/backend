package tools

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func WaitForShutdownSignal(serviceName string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Printf("Shutting down %s...", serviceName)
}
