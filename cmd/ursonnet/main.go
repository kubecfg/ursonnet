package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/google/go-jsonnet"
	"github.com/mkmik/ursonnet"
)

type Context struct {
	*CLI
}

type CLI struct {
	Path      string `arg:""`
	FieldPath string `arg:"" default:"$" help:"jsonnet field path, example, $.a.b"`
	Debug     bool   `short:"d"`
}

func (cmd *CLI) Run(cli *Context) error {
	vm := jsonnet.MakeVM()

	res, err := ursonnet.Roots(vm, cli.Path, cli.FieldPath, ursonnet.Debug(cmd.Debug))
	if err != nil {
		return err
	}
	for _, i := range res {
		fmt.Println(i)
	}
	return nil
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli)
	err := ctx.Run(&Context{CLI: &cli})
	ctx.FatalIfErrorf(err)
}
