package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dyalog-api-go/internal/config"
	apphttp "dyalog-api-go/internal/http"
)

func main() {
	cfg, err := config.Carregar()
	if err != nil {
		log.Fatalf("erro ao carregar configuracao: %v", err)
	}

	servidor, err := apphttp.NovoServidor(cfg)
	if err != nil {
		log.Fatalf("erro ao iniciar servidor: %v", err)
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Porta,
		Handler:           servidor.Engine(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("Dyalog API GO ouvindo na porta %s", cfg.Porta)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("erro no servidor http: %v", err)
		}
	}()

	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, syscall.SIGINT, syscall.SIGTERM)
	<-sinais

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("erro ao encerrar servidor: %v", err)
	}
}
