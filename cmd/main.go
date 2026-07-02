package main

import (
	"context"
	"log"
	"os"

	"github.com/compico/go-osu/cmd/commands"
	"github.com/urfave/cli/v3"

	_ "github.com/compico/go-osu/cmd/commands/http"
)

func main() {
	ctx := context.Background()
	cmd := &cli.Command{
		Commands: commands.Commands,
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		log.Fatalf("%v\n", err)
	}
}
