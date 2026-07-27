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

	args := os.Args
	if len(args) == 1 || (len(args) > 1 && args[1][0] == '-') {
		args = append([]string{args[0], "http"}, args[1:]...)
	}

	cmd := &cli.Command{
		Commands: commands.Commands,
	}

	if err := cmd.Run(ctx, args); err != nil {
		log.Fatalf("%v\n", err)
	}
}
