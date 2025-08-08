package gittest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

type GitServerConfig struct {
	RepoName string
	User     string
	Password string
	Email    string
}

// GitServer represents a Gitea server instance running in a container.
// It provides methods to manage users, repositories, and server operations
// for testing purposes.
type GitServer struct {
	container testcontainers.Container // The container running the Gitea server
	URL       string
	Token     string
}

// NewGitServer creates and initializes a new Gitea server instance in a container.
// It configures the server with default settings and waits for it to be ready.
// The server is configured with:
// - SQLite database
// - Disabled registration
// - Pre-configured admin user
// - Disabled SSH and mailer
// Returns a GitServer instance ready for testing.
func NewGitServer(ctx context.Context, config GitServerConfig) (*GitServer, error) {
	// Configure Gitea container
	req := testcontainers.ContainerRequest{
		Image:        "gitea/gitea:latest",
		ExposedPorts: []string{"3000/tcp"},
		Env: map[string]string{
			"GITEA__database__DB_TYPE":                "sqlite3",
			"GITEA__server__ROOT_URL":                 "http://localhost:3000/",
			"GITEAlogger__server__HTTP_PORT":          "3000",
			"GITEA__service__DISABLE_REGISTRATION":    "true",
			"GITEA__security__INSTALL_LOCK":           "true",
			"GITEA__security__DEFAULT_ADMIN_NAME":     "giteaadmin",
			"GITEA__security__DEFAULT_ADMIN_PASSWORD": "admin123",
			"GITEA__security__SECRET_KEY":             "supersecretkey",
			"GITEA__security__INTERNAL_TOKEN":         "internal",
			"GITEA__security__DISABLE_GITEA_SSH":      "true",
			"GITEA__mailer__ENABLED":                  "false",
			"GITEA__repository__ROOT":                 "/data/git/repositories",
		},
		WaitingFor: wait.ForHTTP("/api/v1/version").WithPort("3000").WithStartupTimeout(30 * time.Second),
	}

	// start container
	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})

	if err != nil {
		return nil, fmt.Errorf("starting gitea container %w", err)
	}

	// get container endpoint
	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting gitserver host %w", err)
	}

	port, err := container.MappedPort(ctx, "3000")
	if err != nil {
		return nil, fmt.Errorf("getting gitserver port %w", err)
	}

	// create user
	err = creteGiteaUser(ctx, container, config.User, config.Email, config.Password)
	if err != nil {
		return nil, fmt.Errorf("creating user %w", err)
	}
	//create repo
	err = createGiteaRepo(ctx, container, config.User, config.Password, config.RepoName)
	if err != nil {
		return nil, fmt.Errorf("creating repo %w", err)
	}

	// generate access token for the user
	token, err := generateToken(ctx, container, config.User)

	return &GitServer{
		URL:       fmt.Sprintf("http://%s:%s/%s/%s.git", host, port.Port(), config.User, config.RepoName),
		Token:     token,
		container: container,
	}, nil
}

func creteGiteaUser(ctx context.Context, container testcontainers.Container, username, email, password string) error {
	_, _, err := container.Exec(context.Background(), []string{
		"su", "git", "-c", fmt.Sprintf("gitea admin user create --username %s --email %s --password %s --must-change-password=false --admin", username, email, password),
	})
	if err != nil {
		return fmt.Errorf("creating user %w", err)
	}
	return nil
}

func createGiteaRepo(ctx context.Context, container testcontainers.Container, user, password, repoName string) error {
	// get container endpoint
	host, err := container.Host(ctx)
	if err != nil {
		return fmt.Errorf("getting gitserver host %w", err)
	}

	port, err := container.MappedPort(ctx, "3000")
	if err != nil {
		return fmt.Errorf("getting gitserver port %w", err)
	}
	createRepoURL := fmt.Sprintf("http://%s:%s/api/v1/user/repos", host, port.Port())
	jsonData := []byte(fmt.Sprintf(`{"name":"%s"}`, repoName))
	reqCreate, err := http.NewRequestWithContext(context.Background(), "POST", createRepoURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating gitea request %w", err)
	}
	reqCreate.Header.Set("Content-Type", "application/json")
	reqCreate.SetBasicAuth(user, password)
	resp, err := http.DefaultClient.Do(reqCreate)
	if err != nil {
		return fmt.Errorf("performing gitea request %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("creating repo status code %s", resp.Status)
	}
	return nil
}

func generateToken(ctx context.Context, container testcontainers.Container, user string) (string, error) {
	token := ""
	cmd := []string{
		"su", "git", "-c", fmt.Sprintf("gitea admin user generate-access-token --username %s --token-name api-token --scopes write:repository,write:user --raw", user),
	}
	exitCode, reader, err := container.Exec(context.Background(), cmd, exec.Multiplexed())
	if err != nil {
		return "", fmt.Errorf("generating access token %w", err)
	}

	// Read the token from the command output
	// TODO: return error if command failed or did not return a token
	if exitCode == 0 && reader != nil {
		tokenBytes, err := io.ReadAll(reader)
		if err == nil && len(tokenBytes) > 0 {
			token = strings.TrimSpace(string(tokenBytes))
		}
	}

	return token, nil
}
