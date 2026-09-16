package create_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"wptui/internal/config"
	"wptui/internal/create"
	"wptui/internal/wpcli"
)

func TestTicket02_DemoBasicProvisioningAndRollback(t *testing.T) {
	tempDir := t.TempDir()
	cfg := config.DefaultConfig(tempDir)
	cfg.WebsitesPath = filepath.Join(tempDir, "sites")

	// 1. Success Demo
	runner := &mockRunner{}
	client := wpcli.NewClientWithRunner(runner)
	creator := create.NewCreator(cfg, client, func(ctx context.Context, dbName string) (bool, error) {
		conn := wpcli.DBConnection{
			Host:   cfg.DatabaseHost,
			Port:   cfg.DatabasePort,
			User:   cfg.DBUsername,
			Pass:   cfg.DBPassword,
			Socket: cfg.DBSocket,
		}
		return client.CheckDatabaseExists(ctx, conn, dbName)
	})

	req := create.Request{
		WebsiteName:   "Coffee Shop",
		WebsiteSlug:   "coffee-shop",
		AdminUsername: "barista",
		AdminPassword: "espresso_password",
		AdminEmail:    "coffee@local.test",
	}

	res, err := creator.Create(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("unexpected create failure: %v", err)
	}

	if res.WebsiteURL != "https://coffee-shop.test" {
		t.Errorf("expected HTTPS .test URL for herd site, got %s", res.WebsiteURL)
	}

	siteDir := filepath.Join(cfg.WebsitesPath, "coffee-shop")
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		t.Errorf("expected website directory to exist at %s", siteDir)
	}

	// 2. Rollback Demo on Install Failure
	failingRunner := &mockRunner{
		failOnSubstr: "wp core install",
	}
	failingClient := wpcli.NewClientWithRunner(failingRunner)
	failingCreator := create.NewCreator(cfg, failingClient, func(ctx context.Context, dbName string) (bool, error) {
		return false, nil
	})

	reqFail := create.Request{
		WebsiteName:   "Doomed Site",
		WebsiteSlug:   "doomed-site",
		AdminUsername: "admin",
		AdminPassword: "password",
		AdminEmail:    "admin@fail.test",
	}

	_, err = failingCreator.Create(context.Background(), reqFail, nil)
	if err == nil {
		t.Fatal("expected error on failing core install, got nil")
	}

	doomedDir := filepath.Join(cfg.WebsitesPath, "doomed-site")
	if _, err := os.Stat(doomedDir); !os.IsNotExist(err) {
		t.Errorf("expected doomed website directory to be rolled back, but it exists")
	}

	// Verify db drop was called during rollback
	dbDropped := false
	for _, call := range failingRunner.calls {
		if strings.Contains(call, "wp db drop") {
			dbDropped = true
			break
		}
	}
	if !dbDropped {
		t.Errorf("expected wp db drop to be invoked during rollback")
	}

	// 3. Database collision demo using CheckDatabaseExists
	collisionRunner := &mockRunner{}
	collisionClient := wpcli.NewClientWithRunner(collisionRunner)
	collisionCreator := create.NewCreator(cfg, collisionClient, func(ctx context.Context, dbName string) (bool, error) {
		if dbName == "existing-db" {
			return true, nil
		}
		return false, nil
	})

	reqCollided := create.Request{
		WebsiteName:   "Existing DB Site",
		WebsiteSlug:   "existing-db",
		AdminUsername: "admin",
		AdminPassword: "password",
		AdminEmail:    "admin@example.com",
	}

	_, err = collisionCreator.Create(context.Background(), reqCollided, nil)
	if err == nil {
		t.Fatal("expected collision error when DB exists, got nil")
	}
	if !create.IsCollisionError(err) {
		t.Errorf("expected CollisionError, got %v", err)
	}

	collidedDir := filepath.Join(cfg.WebsitesPath, "existing-db")
	if _, err := os.Stat(collidedDir); !os.IsNotExist(err) {
		t.Errorf("directory should not have been created on collision")
	}
}
