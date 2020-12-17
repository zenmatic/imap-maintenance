package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
        "github.com/urfave/cli/altsrc"
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
	app.Usage = "purge or sort IMAP mailboxes"
	app.Before = func(ctx *cli.Context) error {
		if ctx.GlobalBool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
	app.Version = VERSION
	app.Author = ""
	app.Email = ""
	flags := []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Debug logging",
		},
		cli.BoolFlag{
			Name:  "dry-run, n",
			Usage: "Dry run mode",
		},
		altsrc.NewStringFlag(cli.StringFlag{
                        Name:  "server",
                        Usage: "imap server",
                }),
		altsrc.NewStringFlag(cli.StringFlag{
			Name:  "port, P",
			Value: "993",
			Usage: "imap server port",
		}),
		altsrc.NewStringFlag(cli.StringFlag{
			Name:  "user",
			Usage: "imap user",
		}),
		altsrc.NewBoolFlag(cli.BoolFlag{
                        Name:  "tls",
                        Usage: "connect with TLS",
                }),
		cli.StringFlag{
			Name:  "config",
			Usage: "load settings from config file `FILE'",
                        Value: os.Getenv("HOME") + "/.imap-maintenance.yml",
		},
	}
        app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("config"))
        app.Flags = flags
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
			Name:        "archive-by-year",
			Usage:       "archive messages into a yearly folder",
			Description: "\nputs messages sent in 2004 into a folder named 2004",
			ArgsUsage:   "None",
			Action:      archiveByYear,
			Flags:       []cli.Flag{
				cli.Int64Flag{
					Name:  "age, A",
					Value: 400,
					Usage: "archive messages older than AGE (in days)",
				},
				cli.IntFlag{
					Name:  "batch, B",
					Value: 25,
					Usage: "purge in batches",
				},
				cli.IntFlag{
					Name:  "max, m",
					Value: 0,
					Usage: "maximum number of batches to run",
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

func connectAndLogin(imapServer string, imapPort string, imapUser string, imapPass string, useTls bool) (*client.Client, error) {
	fullServer := imapServer + ":" + imapPort
	logrus.Infof("Connecting to %s", fullServer)
	var err error
	var c *client.Client
	if useTls {
		logrus.Info("Using TLS...")
		c, err = client.DialTLS(fullServer, nil)
	} else {
		c, err = client.Dial(fullServer)
	}
	if err != nil {
		return c, err
	}
	logrus.Info("Connected")

	logrus.Info("Logging in")
	if err := c.Login(imapUser, imapPass); err != nil {
		return c, err
	}
	logrus.Info("Logged in")

	return c, err
}

func archiveByYear (ctx *cli.Context) error {
	var done chan error
	var messages chan *imap.Message

	msgAge := ctx.Int64("age")
	batch := ctx.Int("batch")
	maxBatches := ctx.Int("max")
	logrus.Debugf("num of args %d", ctx.NArg())
	if ctx.NArg() < 1 {
		logrus.Fatal("no folders passed")
	}
	folders := ctx.Args()

	imapServer := ctx.GlobalString("server")
	imapPort := ctx.GlobalString("port")
	imapUser := ctx.GlobalString("user")
	imapPass, err := promptPassword()
	if err != nil {
		return err
	}

	c, err := connectAndLogin(imapServer, imapPort, imapUser, imapPass, ctx.GlobalBool("tls"))
	if err != nil {
		logrus.Fatal(err)
	}
	defer c.Logout()

	for _, folder := range folders {
		mbox, err := c.Select(folder, false)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Flags for %s: %v", folder, mbox.Flags)
		logrus.Infof("Unread: %d, Total: %d", mbox.Unseen, mbox.Messages)

		lastset := new(imap.SeqSet)
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
		logrus.Debugf("%v", seqNums)
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
                batchct := 0
		for start < seqLen && batchct < maxBatches {
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

                        yearmsgs := make(map[string]*imap.SeqSet)
			for msg := range messages {
				logrus.Infof("* %v %v", msg.Envelope.Date, msg.Envelope.Subject)
                                y := msg.Envelope.Date.Year()
                                // TODO: verify that year is sane
                                logrus.Infof("year is %v", y)
                                ystr := strconv.Itoa(y)
                                if _, ok := yearmsgs[ystr]; !ok {
                                        yearmsgs[ystr] = new(imap.SeqSet)
                                }
                                yearmsgs[ystr].AddNum(msg.SeqNum)
			}

			if err := <-done; err != nil {
				logrus.Fatal(err)
			}
			logrus.Infof("batch %v is done", ct)

                        for yearbox, yearseq := range yearmsgs {
                                logrus.Infof("copying %v to %v", yearseq, yearbox)
                                // TODO: check if mailbox exists, create if not
                                if err := c.Copy(yearseq, yearbox); err != nil {
                                        logrus.Fatal(err)
                                }

                                logrus.Info("done with copy")
                                item := imap.FormatFlagsOp(imap.AddFlags, true)
                                flags := []interface{}{imap.DeletedFlag}
                                if err := c.Store(yearseq, item, flags, nil); err != nil {
                                        logrus.Fatal(err)
                                }

                                if err := c.Expunge(nil); err != nil {
                                        logrus.Fatal(err)
                                }
                        }

			sleeptime := 30 * 1000 * time.Millisecond
			logrus.Infof("sleep %d seconds", sleeptime / 1000*time.Millisecond)
			time.Sleep(sleeptime)

                        batchct++
		}
		logrus.Infof("Done with %s", folder)
	}

	return nil
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

	imapServer := ctx.GlobalString("server")
	imapPort := ctx.GlobalString("port")
	imapUser := ctx.GlobalString("user")
	imapPass, err := promptPassword()
	if err != nil {
		return err
	}

	c, err := connectAndLogin(imapServer, imapPort, imapUser, imapPass, ctx.GlobalBool("tls"))
	if err != nil {
		logrus.Fatal(err)
	}
	defer c.Logout()

	for _, folder := range folders {
		mbox, err := c.Select(folder, false)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Flags for %s: %v", folder, mbox.Flags)
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
		logrus.Debugf("%v", seqNums)
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
