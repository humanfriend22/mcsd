package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "mcsd/utils"
)

// passwordSentinel is written into every fixture that exercises an error
// path so every subtest can assert the credential-leak prohibition (T-01-01)
// from the shared assertion path below, rather than trusting each subtest to
// remember it individually.
const passwordSentinel = "S3ntinel-Rcon-Pass-Do-Not-Leak"

// assertNoCredentialLeak fails the test if err's message contains the
// password sentinel written into the fixture.
func assertNoCredentialLeak(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), passwordSentinel) {
		t.Fatalf("error message leaked the rcon.password sentinel: %s", err.Error())
	}
}

func TestReadPorts(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		dir := t.TempDir()
		contents := "server-port=25565\n" +
			"rcon.port=25575\n" +
			"rcon.password=" + passwordSentinel + "\n"
		if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(contents), 0640); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		ports, err := ReadPorts(dir)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if ports.Game != 25565 {
			t.Fatalf("expected Game 25565, got %d", ports.Game)
		}
		if ports.RCON != 25575 {
			t.Fatalf("expected RCON 25575, got %d", ports.RCON)
		}
		if ports.RCONPassword != passwordSentinel {
			t.Fatalf("expected RCONPassword %q, got %q", passwordSentinel, ports.RCONPassword)
		}
	})

	t.Run("absent file", func(t *testing.T) {
		dir := t.TempDir()

		_, err := ReadPorts(dir)
		if err == nil {
			t.Fatalf("expected an error for a missing server.properties")
		}
		var notFound *NotFoundError
		if !errors.As(err, &notFound) {
			t.Fatalf("expected *NotFoundError, got %T: %v", err, err)
		}
		assertNoCredentialLeak(t, err)
	})

	t.Run("unreadable file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("running as root: a 0000-mode file is still readable, assertion would be invalid")
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "server.properties")
		contents := "server-port=25565\n" +
			"rcon.password=" + passwordSentinel + "\n"
		if err := os.WriteFile(path, []byte(contents), 0640); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		if err := os.Chmod(path, 0000); err != nil {
			t.Fatalf("chmod fixture: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0640) })

		_, err := ReadPorts(dir)
		if err == nil {
			t.Fatalf("expected an error for an unreadable server.properties")
		}
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			t.Fatalf("expected a *ServerError (not *NotFoundError) for a permission failure, got: %v", err)
		}
		var serverErr *ServerError
		if !errors.As(err, &serverErr) {
			t.Fatalf("expected *ServerError, got %T: %v", err, err)
		}
		assertNoCredentialLeak(t, err)
	})

	t.Run("bad server-port", func(t *testing.T) {
		dir := t.TempDir()
		contents := "server-port=not-a-number\n" +
			"rcon.port=25575\n" +
			"rcon.password=" + passwordSentinel + "\n"
		if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(contents), 0640); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		_, err := ReadPorts(dir)
		if err == nil {
			t.Fatalf("expected an error for a non-numeric server-port")
		}
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected *ValidationError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "server-port") {
			t.Fatalf("expected error message to name server-port, got: %s", err.Error())
		}
		if !strings.Contains(err.Error(), "not-a-number") {
			t.Fatalf("expected error message to include the offending value, got: %s", err.Error())
		}
		assertNoCredentialLeak(t, err)
	})

	t.Run("bad rcon.port", func(t *testing.T) {
		dir := t.TempDir()
		contents := "server-port=25565\n" +
			"rcon.port=also-not-a-number\n" +
			"rcon.password=" + passwordSentinel + "\n"
		if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(contents), 0640); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		_, err := ReadPorts(dir)
		if err == nil {
			t.Fatalf("expected an error for a non-numeric rcon.port")
		}
		var valErr *ValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("expected *ValidationError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "rcon.port") {
			t.Fatalf("expected error message to name rcon.port, got: %s", err.Error())
		}
		if !strings.Contains(err.Error(), "also-not-a-number") {
			t.Fatalf("expected error message to include the offending value, got: %s", err.Error())
		}
		assertNoCredentialLeak(t, err)
	})

	t.Run("absent port keys", func(t *testing.T) {
		dir := t.TempDir()
		contents := "rcon.password=" + passwordSentinel + "\n" +
			"online-mode=true\n"
		if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte(contents), 0640); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		ports, err := ReadPorts(dir)
		if err != nil {
			t.Fatalf("expected no error when port keys are absent, got: %v", err)
		}
		if ports.Game != 0 || ports.RCON != 0 {
			t.Fatalf("expected zero-valued ports when keys are absent, got: %+v", ports)
		}
	})
}
