package config

import "testing"

func TestLoadDockerConfigAppliesPublicEnvironmentOverrides(t *testing.T) {
	t.Setenv("ECHOCHAT_SERVER_MODE", "release")
	t.Setenv("ECHOCHAT_SERVER_WS_ALLOWED_ORIGINS", "https://chat.example.test,https://admin.example.test")
	t.Setenv("ECHOCHAT_DATABASE_USER", "public_user")
	t.Setenv("ECHOCHAT_DATABASE_PASSWORD", "public_password")
	t.Setenv("ECHOCHAT_DATABASE_DBNAME", "public_database")
	t.Setenv("ECHOCHAT_REDIS_PASSWORD", "redis_password")
	t.Setenv("ECHOCHAT_MINIO_ACCESS_KEY", "minio_user")
	t.Setenv("ECHOCHAT_MINIO_SECRET_KEY", "minio_password")
	t.Setenv("ECHOCHAT_MINIO_BUCKET", "public_bucket")

	cfg, err := Load(".", "config.docker")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Mode != "release" {
		t.Fatalf("Server.Mode = %q, want release", cfg.Server.Mode)
	}
	if got := cfg.Server.AllowedOrigins(); len(got) != 2 || got[0] != "https://chat.example.test" || got[1] != "https://admin.example.test" {
		t.Fatalf("Server.AllowedOrigins() = %#v", got)
	}
	if cfg.Database.User != "public_user" || cfg.Database.Password != "public_password" || cfg.Database.DBName != "public_database" {
		t.Fatalf("Database overrides not applied: %#v", cfg.Database)
	}
	if cfg.Redis.Password != "redis_password" {
		t.Fatalf("Redis.Password = %q", cfg.Redis.Password)
	}
	if cfg.Minio.AccessKey != "minio_user" || cfg.Minio.SecretKey != "minio_password" || cfg.Minio.Bucket != "public_bucket" {
		t.Fatalf("Minio overrides not applied: %#v", cfg.Minio)
	}
}

func TestLoadDevConfigAppliesWSOriginsOverride(t *testing.T) {
	t.Setenv("ECHOCHAT_SERVER_WS_ALLOWED_ORIGINS", "https://dev.example.test")

	cfg, err := Load(".", "config.dev")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Server.AllowedOrigins(); len(got) != 1 || got[0] != "https://dev.example.test" {
		t.Fatalf("Server.AllowedOrigins() = %#v", got)
	}
}
