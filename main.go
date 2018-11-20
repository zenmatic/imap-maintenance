package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	//"github.com/emersion/go-imap-move"
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
				cli.Int64Flag{
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
		cli.Command{
			Name:        "sort-by-year",
			Usage:       "put old INBOX messages into year folders",
			Description: "\nput old INBOX messages into year folders",
			ArgsUsage:   "None",
			Action:      sortINBOX,
			Flags:       []cli.Flag{
				cli.StringFlag{
					Name:  "inbox",
					Value: "INBOX",
					Usage: "inbox to operate on",
				},
				cli.Int64Flag{
					Name:  "max, m",
					Value: 500,
					Usage: "stop after max operations",
				},
				cli.Int64Flag{
					Name:  "age, A",
					Value: 730,
					Usage: "older than AGE (in days)",
				},
				cli.IntFlag{
					Name:  "batch, B",
					Value: 25,
					Usage: "move messages in in batches",
				},
			},
		},
	}
	// future command: move old user dirs
	return app.Run(os.Args)
}

func promptPassword() (password string, err error) {
	fmt.Print("Password: ")
	bytes, err := terminal.ReadPassword(0)
	password = string(bytes)
	return
}

func connectAndLogin(ctx *cli.Context) (*client.Client, error) {
	imapServer := ctx.GlobalString("server")
	imapPort := ctx.GlobalInt("port")
	imapUser := ctx.GlobalString("user")
	imapPass, err := promptPassword()
	if err != nil {
		return nil, err
	}

	logrus.Info("Connecting to server...")
	fullServer := imapServer + ":" + strconv.Itoa(imapPort)
	c, err := client.DialTLS(fullServer, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info("Connected")

	logrus.Info("Logging in")
	if err := c.Login(imapUser, imapPass); err != nil {
		return c, err
	}
	logrus.Info("Logged in")

	return c, err
}

func purgeFolders(ctx *cli.Context) error {
	var done chan error
	var messages chan *imap.Message

	msgAge := ctx.Int64("age")
	batch := ctx.Int("batch")
	logrus.Debugf("num of args %d", ctx.NArg())
	if ctx.NArg() < 1 {
		logrus.Fatal("no folders passed")
	}
	folders := ctx.Args()

	c, err := connectAndLogin(ctx)
	if err != nil {
		logrus.Fatal(err)
	}
	defer c.Logout()

	for _, folder := range folders {
		mbox, err := c.Select(folder, false)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Flags for %s: %v %-v", folder, mbox.Flags)
		logrus.Infof("Unread: %d, Total: %d", mbox.Unseen, mbox.Messages)

		lastset := new(imap.SeqSet)
		//lastset.AddNum(mbox.Messages)
		lastset.AddNum(1)
		messages = make(chan *imap.Message, 1)
		done = make(chan error, 1)
		go func() {
			done <- c.Fetch(lastset, []imap.FetchItem{imap.FetchEnvelope}, messages)
		}()
		if err := <-done; err != nil {
			logrus.Fatal(err)
		}
		if msg := <-messages; msg != nil {
			logrus.Infof("Oldest message: %v %v", msg.Envelope.Date, msg.Envelope.Subject)
		}

		t := time.Now()
		var day int64
		day = 60*60*24
		beforeTime := t.Unix() - (day*msgAge)
		before := time.Unix(beforeTime, 0)
		searchCrit := new(imap.SearchCriteria)
		searchCrit.Before = before
		seqNums, err := c.Search(searchCrit)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Debug("%v", seqNums)
		for num, _ := range seqNums {
			logrus.Debugf("seqnum: %v", num)
		}

		seqset := new(imap.SeqSet)
		seqLen := len(seqNums)
		start := 0
		end := start + batch
		if seqLen < batch {
			end = seqLen
		}
		ct := 1
		for start < seqLen {
			logrus.Infof("batch %v is %v", ct, seqNums[start:end])
			seqset.Clear()
			seqset.AddNum(seqNums[start:end]...)

			ct++
			start = end
			end = start + batch
			if end > seqLen {
				end = seqLen
			}
			messages = make(chan *imap.Message, end-start)
			done = make(chan error, 1)
			go func() {
				done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
			}()

			for msg := range messages {
				logrus.Infof("* %v %v", msg.Envelope.Date, msg.Envelope.Subject)
			}

			if err := <-done; err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("batch %v is done", ct)

			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.DeletedFlag}
			if err := c.Store(seqset, item, flags, nil); err != nil {
				logrus.Fatal(err)
			}

			if err := c.Expunge(nil); err != nil {
				logrus.Fatal(err)
			}

			sleeptime := 30 * 1000 * time.Millisecond
			logrus.Infof("sleep %d seconds", sleeptime / 1000*time.Millisecond)
			time.Sleep(sleeptime)
		}
		logrus.Infof("Done with %s", folder)
	}

	return nil
}

func sortINBOX(ctx *cli.Context) error {
	/*
	var done chan error
	var messages chan *imap.Message

	batch := ctx.Int("batch")
	*/
	msgAge := ctx.Int64("age")
	folder := ctx.String("inbox")

	c, err := connectAndLogin(ctx)
	if err != nil {
		logrus.Fatal(err)
	}
	defer c.Logout()

	t := time.Now()
	var day int64
	day = 60*60*24
	beforeTime := t.Unix() - (day*msgAge)
	before := time.Unix(beforeTime, 0)
	searchCrit := new(imap.SearchCriteria)
	searchCrit.Before = before

	seqSet, err := filterMessages(c, folder, searchCrit)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("Found the following %v", seqSet)

	return nil
}

func filterMessages(c *client.Client, folder string, filter *imap.SearchCriteria) ([]uint32, error) {
	mbox, err := c.Select(folder, false)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("Flags for %s: %v %-v", folder, mbox.Flags)
	logrus.Infof("Unread: %d, Total: %d", mbox.Unseen, mbox.Messages)

	seqNums, err := c.Search(filter)
	if err != nil {
		logrus.Fatal(err)
	}

	return seqNums, nil
}
