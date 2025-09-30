package digest

import (
	"context"
	"fmt"
)

func Run(ctx context.Context, content string) error {
	fmt.Printf("Digesting %s\n", content)
	return nil
}
