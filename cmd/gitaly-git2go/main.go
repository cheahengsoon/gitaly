//go:build static && system_libgit2
// +build static,system_libgit2

package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"os"

	"gitlab.com/gitlab-org/gitaly/v14/internal/git2go"
)

type subcmd interface {
	Flags() *flag.FlagSet
	Run(ctx context.Context, decoder *gob.Decoder, encoder *gob.Encoder) error
}

var subcommands = map[string]subcmd{
	"apply":       &applySubcommand{},
	"cherry-pick": &cherryPickSubcommand{},
	"commit":      commitSubcommand{},
	"conflicts":   &conflictsSubcommand{},
	"merge":       &mergeSubcommand{},
	"rebase":      &rebaseSubcommand{},
	"revert":      &revertSubcommand{},
	"resolve":     &resolveSubcommand{},
	"submodule":   &submoduleSubcommand{},
}

func fatalf(encoder *gob.Encoder, format string, args ...interface{}) {
	encoder.Encode(git2go.Result{
		Error: git2go.SerializableError(fmt.Errorf(format, args...)),
	})
	os.Exit(1)
}

func main() {
	flags := flag.NewFlagSet(git2go.BinaryName, flag.ExitOnError)

	decoder := gob.NewDecoder(os.Stdin)
	encoder := gob.NewEncoder(os.Stdout)

	if err := flags.Parse(os.Args[1:]); err != nil {
		fatalf(encoder, "parsing flags: %s", err)
	}

	if flags.NArg() < 1 {
		fatalf(encoder, "missing subcommand")
	}

	subcmd, ok := subcommands[flags.Arg(0)]
	if !ok {
		fatalf(encoder, "unknown subcommand: %q", flags.Arg(1))
	}

	subcmdFlags := subcmd.Flags()
	if err := subcmdFlags.Parse(flags.Args()[1:]); err != nil {
		fatalf(encoder, "parsing flags of %q: %s", subcmdFlags.Name(), err)
	}

	if subcmdFlags.NArg() != 0 {
		fatalf(encoder, "%s: trailing arguments", subcmdFlags.Name())
	}

	if err := subcmd.Run(context.Background(), decoder, encoder); err != nil {
		fatalf(encoder, "%s: %s", subcmdFlags.Name(), err)
	}
}
