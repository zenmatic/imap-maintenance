package main

import (
	"context"
	"os"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Test_connectAndLogin(t *testing.T) {
	ctx := context.Background()
	usersFile := os.Getenv("PWD") + "/" + "passwd"
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

