package main

import (
	"context"
	"os"
	"strings"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/docker/go-connections/nat"
)

const (
	imapPort = "10143"
	imapUser = "testuser"
	imapPass = "testing.one.two.three"
)

func startImapContainer(ctx context.Context) (testcontainers.Container, error) {

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	usersFile := wd + "/" + "passwd"
	// convert path for Windows
	if strings.HasPrefix(usersFile, "C:\\") {
		oldFile := strings.Replace(usersFile, "C:\\", "/c/", 1)
		usersFile = oldFile
		usersFile = strings.Replace(usersFile, "\\", "/", -1)
	}
	req := testcontainers.ContainerRequest{
		Image: "docker.io/modularitycontainers/dovecot",
		ExposedPorts: []string{"10143/tcp"},
		Env: map[string]string{
			"DEBUG_MODE": "",
			"MYHOSTNAME": "localhost",
			"PLAIN_AUTH": "yes",
		},
		BindMounts: map[string]string{
			usersFile: "/etc/dovecot/users",
		},
		WaitingFor: wait.NewHostPortStrategy("10143/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started: true,
	})
	if err != nil {
		return container, err
	}

	return container, nil
}

func getIpAndPort(ctx context.Context, container testcontainers.Container) (ip string, port nat.Port, err error) {
	ip, err = container.Host(ctx)
	if err != nil {
		return
	}

	port, err = container.MappedPort(ctx, imapPort)
	if err != nil {
		return
	}

	return
}


func Test_connectAndLogin(t *testing.T) {
	ctx := context.Background()
	container, err := startImapContainer(ctx)
	if err != nil {
		t.Error(err)
	}
	defer container.Terminate(ctx)

	ip, port, err := getIpAndPort(ctx, container)
	if err != nil {
		t.Error(err)
	}

	_, err = connectAndLogin(ip, port.Port(), imapUser, imapPass, false)
	if err != nil {
		t.Error(err)
	}
}

func Test_purgeFolder(t *testing.T) {
	ctx := context.Background()
	container, err := startImapContainer(ctx)
	if err != nil {
		t.Error(err)
	}
	defer container.Terminate(ctx)

	ip, port, err := getIpAndPort(ctx, container)
	if err != nil {
		t.Error(err)
	}

	c, err := connectAndLogin(ip, port.Port(), imapUser, imapPass, false)
	if err != nil {
		t.Error(err)
	}

	err = purgeFolder(c, "INBOX", 10, 1)
	if err != nil {
		t.Error(err)
	}

}
