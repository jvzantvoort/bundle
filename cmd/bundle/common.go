package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func GetString(cmd cobra.Command, name string) string {
	retv, _ := cmd.Flags().GetString(name)
	if len(retv) != 0 {
		log.Debugf("%s returned %s", name, retv)
	} else {
		log.Debugf("%s returned nothing", name)
	}
	return retv
}

func handleLogCmd(cmd *cobra.Command, args []string) {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	log.Debugf("%s: start", cmd.Use)
	defer log.Debugf("%s: end", cmd.Use)

	if len(args) == 0 {
		log.Error("No message provided")
		if err := cmd.Help(); err != nil {
			log.Error(err)
		}
		os.Exit(1)
	}
	/*
	logmsg := logging.NewLogMessage(cmd.Use)
	logmsg.ImportArgs(cmd, args)
	err := logmsg.Print()
	if err != nil {
		log.Error(err)

	}*/
}
