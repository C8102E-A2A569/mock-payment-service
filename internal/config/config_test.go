package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_emptyPath_returnsDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\"): %v", err)
	}
	if cfg.Server.GRPCPort != 50051 {
		t.Errorf("default GRPCPort = %d, want 50051", cfg.Server.GRPCPort)
	}
	if cfg.DB.Host != "localhost" || cfg.DB.Port != 5432 {
		t.Errorf("default DB: %s %d", cfg.DB.Host, cfg.DB.Port)
	}
	if len(cfg.Kafka.Brokers) == 0 || cfg.Kafka.TopicPaymentEvents == "" {
		t.Error("default Kafka should be set")
	}
	if cfg.Redis.Addr == "" {
		t.Error("default Redis.Addr should be set")
	}
}

func TestLoad_nonexistentFile_error(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoad_validYaml(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(f, []byte(`
server:
  grpc_port: 9090
db:
  host: dbhost
  port: 5433
`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(f)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.GRPCPort != 9090 {
		t.Errorf("grpc_port = %d, want 9090", cfg.Server.GRPCPort)
	}
	if cfg.DB.Host != "dbhost" || cfg.DB.Port != 5433 {
		t.Errorf("db: %s %d", cfg.DB.Host, cfg.DB.Port)
	}
}

func TestLoad_envOverride(t *testing.T) {
	os.Setenv("GRPC_PORT", "7000")
	os.Setenv("DB_HOST", "envhost")
	defer os.Unsetenv("GRPC_PORT")
	defer os.Unsetenv("DB_HOST")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.GRPCPort != 7000 {
		t.Errorf("GRPC_PORT override: got %d", cfg.Server.GRPCPort)
	}
	if cfg.DB.Host != "envhost" {
		t.Errorf("DB_HOST override: got %s", cfg.DB.Host)
	}
}

func TestDBConfig_DSN(t *testing.T) {
	c := DBConfig{
		User: "u", Password: "p", Host: "h", Port: 5432, DBName: "d", SSLMode: "disable",
	}
	dsn := c.DSN()
	if dsn == "" || !strings.HasPrefix(dsn, "postgres://") {
		t.Errorf("DSN should be postgres URL: %s", dsn)
	}
}
