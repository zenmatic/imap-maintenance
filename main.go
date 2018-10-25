package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap"
	"github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

var VERSION = "0.1"

func main() {
	if err := mainErr(); err != nil {
		logrus.Fatal(err)
	}
}

func mainErr() error {
	app := cli.NewApp()
	app.Name = "imap-maintenance"
	app.Usage = "Rancher CLI, managing containers one UTF-8 character at a time"
	app.Before = func(ctx *cli.Context) error {
		if ctx.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Version = VERSION
	app.Author = ""
	app.Email = ""
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Debug logging",
		},
		cli.BoolFlag{
			Name:  "dry-run, n",
			Usage: "Dry run mode",
		},
		cli.StringFlag{
			Name:  "server, s",
			Usage: "imap server",
		},
		cli.IntFlag{
			Name:  "port, P",
			Value: 993,
			Usage: "imap server port",
		},
		cli.StringFlag{
			Name:  "user, u",
			Usage: "imap user",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Name:        "purge",
			Usage:       "purge folders",
			Description: "\npurge IMAP folders based on age",
			ArgsUsage:   "None",
			Action:      purgeFolders,
			Flags:       []cli.Flag{
				cli.IntFlag{
					Name:  "age, A",
					Value: 400,
					Usage: "older than AGE (in days)",
				},
				cli.IntFlag{
					Name:  "batch, B",
					Value: 25,
					Usage: "purge in batches",
				},
			},
		},
	}
	// future command: sort by year
	// future command: move old user dirs
	return app.Run(os.Args)
}

func promptPassword() (password string, err error) {
	fmt.Print("Password: ")
	bytes, err := terminal.ReadPassword(0)
	password = string(bytes)
	return
}

func purgeFolders(ctx *cli.Context) error {
	imapServer := ctx.GlobalString("server")
	imapPort := ctx.GlobalInt("port")
	imapUser := ctx.GlobalString("user")
	imapPass, err := promptPassword()
	if err != nil {
		logrus.Fatal(err)
	}
	//msgAge := ctx.Int("age")
	//batch := ctx.Int("batch")
	logrus.Debugf("num of args %d", ctx.NArg())
	if ctx.NArg() < 1 {
		logrus.Fatal("no folders passed")
	}
	folders := ctx.Args()

	logrus.Info("Connecting to server...")

	// Connect to server
	fullServer := imapServer + ":" + strconv.Itoa(imapPort)
	c, err := client.DialTLS(fullServer, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Connected")

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(imapUser, imapPass); err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Logged in")

	folder := folders[0]
	mbox, err := c.Select(folder, false)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("Flags for %s: %v", folder, mbox.Flags)

	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > 0 {
		// We're using unsigned integers here, only substract if the result is > 0
		from = mbox.Messages - 3
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	logrus.Info("Last 4 messages:")
	for msg := range messages {
		logrus.Info("* " + msg.Envelope.Subject)
	}

	if err := <-done; err != nil {
		logrus.Fatal(err)
	}

	logrus.Info("Done!")
	return nil
}
