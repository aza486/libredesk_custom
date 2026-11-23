package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Re-execute the actual main/flag entry point in a child. No alternate installer
// and no dependency on a manually built binary or frontend build is introduced.
func TestInstallationCLIProcess(t *testing.T) {
	if os.Getenv("LIBREDESK_INSTALL_TEST_CHILD") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"libredesk"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	os.Exit(2)
}

func TestInstallationCLIChain(t *testing.T) {
	db, _ := installationFixture(t, "install_cli")
	dsn := os.Getenv("LIBREDESK_TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://libredesk:libredesk@127.0.0.1:5433/libredesk?sslmode=disable"
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	var dbName string
	if err := db.Get(&dbName, `SELECT current_database()`); err != nil {
		t.Fatal(err)
	}
	host := u.Query().Get("host")
	if host == "" {
		host = u.Hostname()
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	password, _ := u.User.Password()
	if password == "" {
		password = "synthetic-only"
	}
	sample, err := os.ReadFile("../config.sample.toml")
	if err != nil {
		t.Fatal(err)
	}
	config := string(sample)
	a, b := strings.Index(config, "[db]"), strings.Index(config, "[redis]")
	config = config[:a] + fmt.Sprintf("[db]\nhost=%s\nport=%s\nuser=%s\npassword=%s\ndatabase=%s\nssl_mode=\"disable\"\nmax_open=2\nmax_idle=1\nmax_lifetime=\"60s\"\n\n", strconv.Quote(host), port, strconv.Quote(u.User.Username()), strconv.Quote(password), strconv.Quote(dbName)) + config[b:]
	dir := t.TempDir()
	path := filepath.Join(dir, "isolated.toml")
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	run := func(wantSuccess bool, args ...string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestInstallationCLIProcess$", "--", "--config", path}, args...)...)
		cmd.Dir = root
		for _, v := range os.Environ() {
			if !strings.HasPrefix(v, "LIBREDESK_") {
				cmd.Env = append(cmd.Env, v)
			}
		}
		cmd.Env = append(cmd.Env, "LIBREDESK_INSTALL_TEST_CHILD=1", "LIBREDESK_SYSTEM_USER_PASSWORD=Synthetic-CLI-47!Password")
		out, err := cmd.CombinedOutput()
		if (err == nil) != wantSuccess {
			t.Fatalf("CLI %v: %v\n%s", args, err, out)
		}
	}
	parallelInstalls := func() {
		commands := []*exec.Cmd{}
		for range 3 {
			cmd := exec.Command(os.Args[0], "-test.run=^TestInstallationCLIProcess$", "--", "--config", path, "--install", "--idempotent-install", "--yes")
			cmd.Dir = root
			for _, v := range os.Environ() {
				if !strings.HasPrefix(v, "LIBREDESK_") {
					cmd.Env = append(cmd.Env, v)
				}
			}
			cmd.Env = append(cmd.Env, "LIBREDESK_INSTALL_TEST_CHILD=1")
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			commands = append(commands, cmd)
		}
		for _, cmd := range commands {
			if err := cmd.Wait(); err != nil {
				t.Fatal("parallel CLI install failed", err)
			}
		}
	}
	// Race the very first installation across independent processes.
	parallelInstalls()
	run(true, "--install", "--idempotent-install", "--yes")
	if err := checkInstalledDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	before := installationSnapshot(t, db)
	run(true, "--upgrade", "--yes")
	run(true, "--install", "--idempotent-install", "--yes")
	run(true, "--check-system-tags")
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("actual CLI chain changed data on repetition")
	}
	parallelInstalls()
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("parallel CLI installs changed data")
	}
	db.MustExec(`ALTER TABLE users DROP COLUMN work_phone_number`)
	run(false, "--check-system-tags")
	run(false) // Normal main must fail before initializing services.
}
