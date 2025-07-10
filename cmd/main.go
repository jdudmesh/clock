package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"clock/almanac"
	"clock/router"
	"clock/temperature"

	"github.com/rs/zerolog"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	temperature := temperature.New(&logger)
	almanac := almanac.New(&logger)

	go temperature.Run(ctx)
	go almanac.Run(ctx)

	mux := router.NewRouter(&logger, temperature, almanac)
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
		logger.Info().Msg("server stopped")
	}()

	logger.Info().Msg("starting server")
	err := server.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("failed to start server")
			os.Exit(1)
		}
	}
}
