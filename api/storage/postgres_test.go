package storage_test

import (
	"api/domain"
	"api/storage"
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

var repo *storage.PostgresPlayerRepo

func TestMain(m *testing.M) {
	ctx := context.Background()
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	absPath := filepath.Join(pwd, "../../postgres")

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine3.22",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testusername"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			hostConfig.Binds = append(hostConfig.Binds, absPath+":/docker-entrypoint-initdb.d")
		}),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		panic(err)
	}

	connString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	repo, err = storage.NewPostgresRepo(ctx, connString)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	// Cleanup
	postgresContainer.Terminate(ctx)
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
		_, err := repo.CreateUser(ctx, "oussama", "new_hash")
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
		id, err := repo.CreateUser(ctx, "tester2", "hash2")
		require.NoError(t, err)

		user, err := repo.GetUserById(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, "hash2", user.PasswordHash)
		assert.Equal(t, "tester2", user.Username)
	})
}
