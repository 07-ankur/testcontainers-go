package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

type user struct {
	ID    int `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func TestE2E(t *testing.T) {
	ctx := context.Background()

	// 1. A private network for the containers to communicate with each other.
	net, err := network.New(ctx)
	if err != nil {
		t.Fatalf("failed to create network: %v", err)
	}
	t.Cleanup(func() {
		_ = net.Remove(ctx)
	})

	// 2. Start a PostgreSQL container.
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("appdb"),
		postgres.WithUsername("app"),
		postgres.WithPassword("secret"),
		network.WithNetwork([]string{"db"}, net),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start PostgreSQL container: %v", err)
	}
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	// 3. Start the server container.
	serverContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "..",
				Dockerfile: "Dockerfile",
			},
			ExposedPorts: []string{"8080/tcp"},
			Networks:    []string{net.Name},
			Env: map[string]string{
				//Host "db" is the hostname of the PostgreSQL container in the private network.
				"DATABASE_URL": "postgres://app:secret@db:5432/appdb?sslmode=disable",
			},
			WaitingFor: wait.ForHTTP("/healthz").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("failed to start server container: %v", err)
	}
	t.Cleanup(func() {
		_ = serverContainer.Terminate(ctx)
	})

	// 4. Get the server's host and port.
	host, err := serverContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get server host: %v", err)
	}
	port, err := serverContainer.MappedPort(ctx, "8080")
	if err != nil {
		t.Fatalf("failed to get server port: %v", err)
	}
	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	// 5. POST /users - Create a new user.
	resp, err := http.Post(baseURL+"/users", "application/json", bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com"}`))
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	defer resp.Body.Close()
	var createdUser user
	if err := json.NewDecoder(resp.Body).Decode(&createdUser); err != nil {
		t.Fatalf("failed to decode create user response: %v", err)
	}
	if createdUser.ID == 0 {
		t.Fatalf("expected created user to have an ID, got 0")
	}

	// 6. GET /users/{id} - Retrieve the created user.
	getResp, err := http.Get(fmt.Sprintf("%s/users/%d", baseURL, createdUser.ID))
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", getResp.StatusCode)
	}
	var retrievedUser user
	if err := json.NewDecoder(getResp.Body).Decode(&retrievedUser); err != nil {
		t.Fatalf("failed to decode retrieve user response: %v", err)
	}
	if retrievedUser.ID != createdUser.ID {
		t.Fatalf("expected user ID %d, got %d", createdUser.ID, retrievedUser.ID)
	}

	t.Logf("Successfully created and retrieved user: %+v", retrievedUser)
}