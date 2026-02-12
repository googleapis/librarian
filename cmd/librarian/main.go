// Binary surfer ... (optional comment)
// glaze_kind=go_binary
package main

import (
	"context"
	"log"
	"os"

	"github.com/googleapis/librarian/internal/librarian"
)

func main() {
	ctx := context.Background()
	if err := librarian.Run(ctx, os.Args...); err != nil {
		log.Fatalf("librarian: %v", err)
	}
}
