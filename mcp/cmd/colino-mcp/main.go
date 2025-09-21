package main

import (
    "context"
    "log"

    "colino-mcp/internal/server"
)

func main() {
    if err := server.Run(context.Background()); err != nil {
        log.Fatalf("mcp server: %v", err)
    }
}
