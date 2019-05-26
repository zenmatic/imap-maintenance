package main

import (
	"context"
	"os"
	"strings"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_connectAndLogin(t *testing.T) {
	ctx := context.Background()

	wd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}
	usersFile := wd + "/" + "passwd"
	t.Logf("usersFile is %s", usersFile)
	// convert path for Windows
	if strings.HasPrefix(usersFile, "C:\\") {
		oldFile := strings.Replace(usersFile, "C:\\", "/c/", 1)
		usersFile = oldFile
		usersFile = strings.Replace(usersFile, "\\", "/", -1)
		t.Logf("changed usersFile from %s to %s", oldFile, usersFile)
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
		t.Error(err)
	}
	defer container.Terminate(ctx)

	ip, err := container.Host(ctx)
	if err != nil {
		t.Error(err)
	}

	port, err := container.MappedPort(ctx, "10143")
	if err != nil {
		t.Error(err)
	}

	_, err = connectAndLogin(ip, port.Port(), "testuser", "testing.one.two.three", false)
	if err != nil {
		t.Error(err)
	}

}

