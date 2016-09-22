package main

import (
	"os"

	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()

	app.Name = "stern"
	app.Usage = "Tail multiple pods and containers from Kubernetes"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{}
	app.Action = tailAction

	app.Run(os.Args)
}

var tailAction = func(c *cli.Context) error {
	return nil
}
