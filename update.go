package main

import (
	"github.com/dunkelstern/dlite/utils"
)

type UpdateCommand struct {
	Version string `short:"v" long:"version" description:"version of DhyveOS to install"`
}

func (c *UpdateCommand) Execute(args []string) error {
	steps := utils.Steps{
		{
			"Downloading OS",
			func() error {
				if c.Version == "" {
					latest, err := utils.GetLatestOSVersion()
					if err != nil {
						return err
					}
					c.Version = latest
				}
				return utils.DownloadOS(c.Version)
			},
		},
	}

	return utils.Spin(steps)
}

func init() {
	var updateCommand UpdateCommand
	cmd.AddCommand("update", "update your vm", "updates the OS powering your vm", &updateCommand)
}
