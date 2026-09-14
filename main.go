// ynab is a command line for a YNAB plan.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jmcampanini/ynab-cli/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cmd.NewRoot().ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ynab: %s\n", err)
		os.Exit(cmd.ExitCode(err))
	}
}
