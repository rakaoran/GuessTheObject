package database_test

import (
	"api/database"
	"api/domain"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var repo *database.PostgresPlayerRepo

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. FIX: Resolve absolute path for Docker
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Adjust this Join to match where your folder is relative to THIS test file.
	// If your test is in `internal/db` and sql is in `internal/postgres`, use "../postgres"
	absPath := filepath.Join(pwd, "../../postgres")

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine3.22",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testusername"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			// 2. FIX: Use absolute path
			hostConfig.Binds = append(hostConfig.Binds, absPath+":/docker-entrypoint-initdb.d")
		}),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		panic(err)
	}

	// REMOVED: err = postgresContainer.Start(ctx) <-- "Run" already did this!

	connString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	// Initialize your global repo variable
	repo, err = database.NewPostgresRepo(ctx, connString)
	if err != nil {
		panic(err)
	}

	// 3. FIX: Capture exit code
	code := m.Run()

	// Cleanup
	postgresContainer.Terminate(ctx)

	// 4. FIX: Report pass/fail to the OS
	os.Exit(code)
}

func TestPostgresRepo(t *testing.T) {
	ctx := context.Background()
	t.Run("CreateUser", func(t *testing.T) {
		id, err := repo.CreateUser(ctx, "oussama", "hashed_secret")
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("CreateUser_Duplicate", func(t *testing.T) {
		// Try to create 'oussama' again (already created in previous step)
		_, err := repo.CreateUser(ctx, "oussama", "new_hash")

		// Verify that it correctly maps the Unique Violation error to your domain error
		assert.ErrorIs(t, err, domain.ErrDuplicateUsername)
	})

	t.Run("GetUserByUsername", func(t *testing.T) {
		user, err := repo.GetUserByUsername(ctx, "oussama")
		assert.NoError(t, err)
		assert.Equal(t, "oussama", user.Username)
		assert.Equal(t, "hashed_secret", user.PasswordHash)
		assert.NotEmpty(t, user.Id)
	})

	t.Run("GetUserByUsername_NotFound", func(t *testing.T) {
		_, err := repo.GetUserByUsername(ctx, "ghost_user")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})

	t.Run("GetUserById", func(t *testing.T) {
		// First create a fresh user to get a known ID
		id, err := repo.CreateUser(ctx, "tester2", "hash2")
		require.NoError(t, err)

		user, err := repo.GetUserById(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, "hash2", user.PasswordHash)
		assert.Equal(t, "tester2", user.Username)
	})
}
