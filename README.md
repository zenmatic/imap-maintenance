# imap-maintenance

IMAP mailbox maintenance script written for fun in Go.  This is useful if you need to periodically purge or sort IMAP mailboxes.

## build

```go build```

## test

This will fire up up a docker container to simulate a real IMAP server to test against.  So you'll need a docker install for this to work.

```go test```

## usage

```$ ./imap-maintenance.exe
NAME:
   imap-maintenance - purge or sort IMAP mailboxes

USAGE:
   imap-maintenance.exe [global options] command [command options] [arguments...]

VERSION:
   0.1

COMMANDS:
     purge    purge folders
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d               Debug logging
   --dry-run, -n             Dry run mode
   --server value, -s value  imap server
   --port value, -P value    imap server port (default: "993")
   --user value, -u value    imap user
   --tls                     connect with TLS
   --help, -h                show help
   --version, -v             print the version
   ```
   
   ### purge messages older than a certain date
   
   ```$ ./imap-maintenance.exe purge --help
NAME:
   imap-maintenance.exe purge - purge folders

USAGE:
   imap-maintenance.exe purge [command options] None

DESCRIPTION:

purge IMAP folders based on age

OPTIONS:
   --age value, -A value    older than AGE (in days) (default: 400)
   --batch value, -B value  purge in batches (default: 25)
```

#### example: purge messages older than 90 days

``` ./imap-maintenance.exe purge --server imap.gmail.com --user yourgmailuser --tls purge --age 90 bigmailbox```
