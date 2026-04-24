package main

import (
	"flag"
	"gemini-freefall/internal/proxy"
	"log/slog"
	"os"
)

func main() {
	openaiMode := flag.Bool("openai", false, "Enable OpenAI-compatible routing mode")
	flag.Parse()

	cfg, err := proxy.LoadConfig("config.yaml", *openaiMode)
	if err != nil {
		slog.Error("failed to load config", slog.Any("err", err))
		os.Exit(1)

		return
	}

	l := proxy.New(cfg)
	slog.Info("starting proxy server", slog.Any("port", cfg.Config.Port))
	if err := l.ListenAndServe(); err != nil {
		slog.Error("failed to start proxy server", slog.Any("err", err))
		os.Exit(1)

		return
	}
}
