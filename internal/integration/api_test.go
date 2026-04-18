//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/wassiliy/subscriptions-service/internal/db"
	"github.com/wassiliy/subscriptions-service/internal/domain"
	"github.com/wassiliy/subscriptions-service/internal/handler"
	"github.com/wassiliy/subscriptions-service/internal/repository"
	"github.com/wassiliy/subscriptions-service/internal/service"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for range 15 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "go.mod not found from %s", wd)
		dir = parent
	}
	t.Fatalf("go.mod not found")
	return ""
}

func setupServer(t *testing.T) (_ *httptest.Server, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	var connStr string
	var terminate func()

	if envURL := os.Getenv("TEST_DATABASE_URL"); envURL != "" {
		connStr = envURL
		terminate = func() {}
	} else {
		pgCtr, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("subscriptions"),
			postgres.WithUsername("app"),
			postgres.WithPassword("app"),
			postgres.BasicWaitStrategies(),
		)
		if err != nil {
			t.Skipf("postgres container: %v — set TEST_DATABASE_URL or start Docker", err)
		}

		var errC error
		connStr, errC = pgCtr.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, errC)

		terminate = func() {
			if err := testcontainers.TerminateContainer(pgCtr); err != nil {
				t.Logf("terminate postgres: %v", err)
			}
		}
	}

	root := moduleRoot(t)
	migs := filepath.Join(root, "migrations")
	migrationsPath := "file://" + filepath.ToSlash(filepath.Clean(migs))
	require.NoError(t, db.RunMigrations(migrationsPath, connStr))

	pool, err := db.NewPool(ctx, connStr, 30, time.Second)
	require.NoError(t, err)

	repo := repository.NewSubscription(pool)
	log := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	svc := service.NewSubscription(repo, log)
	h := handler.New(svc)

	router := handler.NewRouter(log, h, handler.RateLimitRouterConfig{Enabled: false})
	ts := httptest.NewServer(router)

	cleanup = func() {
		ts.Close()
		pool.Close()
		terminate()
	}
	return ts, cleanup
}

func TestHealth(t *testing.T) {
	ts, cleanup := setupServer(t)
	defer cleanup()

	res, err := http.Get(ts.URL + "/healthz")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	body, _ := io.ReadAll(res.Body)
	require.Equal(t, "ok", strings.TrimSpace(string(body)))
}

func TestSubscriptionCRUDAndCost(t *testing.T) {
	ts, cleanup := setupServer(t)
	defer cleanup()

	client := ts.Client()

	userID := uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba")

	createBody := domain.CreateSubscription{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      userID,
		StartDate:   "07-2025",
		EndDate:     ptr("12-2025"),
	}
	raw, err := json.Marshal(createBody)
	require.NoError(t, err)

	postRes, err := client.Post(ts.URL+"/api/v1/subscriptions", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer postRes.Body.Close()
	require.Equal(t, http.StatusCreated, postRes.StatusCode)

	var created domain.Subscription
	require.NoError(t, json.NewDecoder(postRes.Body).Decode(&created))
	require.NotEqual(t, uuid.Nil, created.ID)

	getRes, err := client.Get(ts.URL + "/api/v1/subscriptions/" + created.ID.String())
	require.NoError(t, err)
	defer getRes.Body.Close()
	require.Equal(t, http.StatusOK, getRes.StatusCode)

	updBody := createBody
	updBody.Price = 500
	raw2, err := json.Marshal(updBody)
	require.NoError(t, err)

	reqPut, err := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/subscriptions/"+created.ID.String(), bytes.NewReader(raw2))
	require.NoError(t, err)
	reqPut.Header.Set("Content-Type", "application/json")

	putRes, err := client.Do(reqPut)
	require.NoError(t, err)
	defer putRes.Body.Close()
	require.Equal(t, http.StatusOK, putRes.StatusCode)

	costURL := ts.URL + "/api/v1/subscriptions/cost?from=07-2025&to=12-2025&user_id=" + userID.String()
	costRes, err := client.Get(costURL)
	require.NoError(t, err)
	defer costRes.Body.Close()
	require.Equal(t, http.StatusOK, costRes.StatusCode)

	var report domain.CostReport
	require.NoError(t, json.NewDecoder(costRes.Body).Decode(&report))
	// 6 months * 500 rub
	require.Equal(t, int64(3000), report.TotalRub)

	listURL := ts.URL + "/api/v1/subscriptions?user_id=" + userID.String() + "&service_name=Yandex%20Plus"
	listRes, err := client.Get(listURL)
	require.NoError(t, err)
	defer listRes.Body.Close()
	require.Equal(t, http.StatusOK, listRes.StatusCode)

	var list []domain.Subscription
	require.NoError(t, json.NewDecoder(listRes.Body).Decode(&list))
	require.Len(t, list, 1)

	reqDel, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/subscriptions/"+created.ID.String(), nil)
	require.NoError(t, err)
	delRes, err := client.Do(reqDel)
	require.NoError(t, err)
	defer delRes.Body.Close()
	require.Equal(t, http.StatusNoContent, delRes.StatusCode)

	get404, err := client.Get(ts.URL + "/api/v1/subscriptions/" + created.ID.String())
	require.NoError(t, err)
	defer get404.Body.Close()
	require.Equal(t, http.StatusNotFound, get404.StatusCode)
}

func ptr(s string) *string { return &s }
