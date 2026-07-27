package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/compico/go-osu/cmd/commands"
	"github.com/urfave/cli/v3"

	_ "github.com/compico/go-osu/cmd/commands/http"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	ctx := context.Background()

	cli.VersionPrinter = func(cmd *cli.Command) {
		fmt.Printf(
			"goosu %s\nbuilt %s\n",
			cmd.Root().Version,
			BuildTime,
		)
	}

	args := os.Args
	if len(args) == 1 {
		args = append(args, "http")
	}

	cmd := &cli.Command{
		Commands: commands.Commands,
		Version:  Version,
	}

	if err := cmd.Run(ctx, args); err != nil {
		log.Fatalf("%v\n", err)
	}
}
