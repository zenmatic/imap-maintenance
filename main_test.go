package main

import (
	"context"
	"testing"

	testcontainers "github.com/testcontainers/testcontainers-go"
)

/* TODO: switch to:
	docker run -it -e MYHOSTNAME=localhost \
	-e DEBUG_MODE \
	-v password=/etc/passwd \
	-v shadow=/etc/shadow  \
	modularitycontainers/dovecot 
*/


func Test_connectAndLogin(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "camptocamp/courier-imap",
		ExposedPorts: []string{"143/tcp"},
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

	port, err := container.MappedPort(ctx, "143")
	if err != nil {
		t.Error(err)
	}

	_, err = connectAndLogin(ip, port.Port(), "smtp", "smtp", false)
	if err != nil {
		t.Error(err)
	}

}

