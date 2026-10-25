// ynab is a command line for a YNAB plan.
package main

import (
	"os"

	"github.com/jmcampanini/ynab-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
