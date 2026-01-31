package storage_test

import (
	"api/domain"
	"api/migrations"
	"api/storage"
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var repo *storage.PostgresRepo

func TestMain(m *testing.M) {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine3.22",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testusername"),
		postgres.WithPassword("testpassword"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		panic(err)
	}

	connString, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	migrations.Migrate(connString)

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

func TestGenerate(t *testing.T) {
	ctx := context.Background()

	t.Run("Generate returns random words", func(t *testing.T) {
		count := 5
		words := repo.Generate(count)

		assert.Len(t, words, count, "Should return requested number of words")

		// Verify words are unique
		wordSet := make(map[string]bool)
		for _, word := range words {
			assert.False(t, wordSet[word], "Words should be unique: %s", word)
			assert.NotEmpty(t, word, "Words should not be empty")
			wordSet[word] = true
		}
	})

	t.Run("Generate with count of 3", func(t *testing.T) {
		words := repo.Generate(3)
		assert.Len(t, words, 3)
	})

	t.Run("Generate with count of 1", func(t *testing.T) {
		words := repo.Generate(1)
		assert.Len(t, words, 1)
	})

	t.Run("Generate with count of 0 returns empty slice", func(t *testing.T) {
		words := repo.Generate(0)
		assert.Empty(t, words)
	})

	t.Run("Generate multiple times gives different results", func(t *testing.T) {
		words1 := repo.Generate(5)
		words2 := repo.Generate(5)

		assert.Len(t, words1, 5)
		assert.Len(t, words2, 5)

		allWords, err := getAllWords(ctx)
		require.NoError(t, err)

		for _, word := range words1 {
			assert.Contains(t, allWords, word, "Word should be from database")
		}
		for _, word := range words2 {
			assert.Contains(t, allWords, word, "Word should be from database")
		}
	})

	t.Run("Generate more than available words", func(t *testing.T) {
		words := repo.Generate(1000)

		assert.NotEmpty(t, words)
		assert.LessOrEqual(t, len(words), 1000)
	})
}

func getAllWords(ctx context.Context) ([]string, error) {
	query := "SELECT word FROM words"
	rows, err := repo.GetPool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	return words, rows.Err()
}
