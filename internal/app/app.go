// Package app is the composition root: it wires configuration, adapters, tools,
// agents, services, and the HTTP server together and runs the service until the
// context is cancelled.
package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"miss-raspberry-agent/internal/adapter/http"
	httphandler "miss-raspberry-agent/internal/adapter/http/handler"
	"miss-raspberry-agent/internal/adapter/napcat"
	mainagent "miss-raspberry-agent/internal/agent/main"
	"miss-raspberry-agent/internal/config"
	"miss-raspberry-agent/internal/llm"
	"miss-raspberry-agent/internal/messaging"
	"miss-raspberry-agent/internal/tagging"
)

// Run builds the whole application from cfg and runs it until ctx is canceled.
func Run(ctx context.Context, cfg config.Config) error {
	chatModel, err := llm.NewChatModel(ctx, cfg.Model)
	if err != nil {
		return fmt.Errorf("construct chat model: %w", err)
	}

	client := napcat.NewClient(&napcat.NapcatClientConfig{
		WebSocketURL:  cfg.Napcat.WebSocketURL,
		AccessToken:   cfg.Napcat.AccessToken,
		NickName:      []string{"bot"},
		CommandPrefix: "/",
		SuperUsers:    []int64{},
	})

	agent, err := mainagent.NewMainAgent(ctx, chatModel, client, client)
	if err != nil {
		return fmt.Errorf("build main agent: %w", err)
	}

	// Route qualifying QQ messages into the agent's own todo queue before the client starts,
	// so no message is dropped during startup.
	client.SetTodoList(agent.Queue())
	if err := client.Start(); err != nil {
		return fmt.Errorf("start napcat client: %w", err)
	}
	defer client.Stop()

	// Expose the HTTP API that lets callers push messages into the same todo queue.
	messageService := messaging.NewService(agent.Queue())
	taggingService := tagging.NewService(tagging.NewStore())
	router := http.NewRouter(
		httphandler.NewMessageHandler(messageService),
		httphandler.NewTaggingHandler(taggingService),
		cfg.HTTP.APIToken,
	)
	server := http.NewServer(cfg.HTTP.Addr, router)

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("[main] HTTP API listening on %s", cfg.HTTP.Addr)
		serverErr <- server.Start()
	}()

	agentDone := make(chan struct{})
	go func() {
		agent.Run(ctx)
		close(agentDone)
	}()

	log.Println("[main] main_agent started, polling its todo queue...")

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-agentDone:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[main] http server shutdown: %v", err)
	}
	if err := <-serverErr; err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}
