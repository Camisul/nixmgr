package nix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldHost(t *testing.T) {
	dir := t.TempDir()
	hostsDir := filepath.Join(dir, "hosts")

	if err := ScaffoldHost(hostsDir, "zeus"); err != nil {
		t.Fatalf("ScaffoldHost: %v", err)
	}

	configPath := filepath.Join(hostsDir, "zeus", "configuration.nix")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `networking.hostName = "zeus"`) {
		t.Errorf("expected hostname in config, got:\n%s", content)
	}
	if !strings.Contains(content, "../../modules/base.nix") {
		t.Errorf("expected base.nix import, got:\n%s", content)
	}
}

func TestAddFlakeOutput(t *testing.T) {
	dir := t.TempDir()
	flakePath := filepath.Join(dir, "flake.nix")

	initial := `{
  outputs = { self, nixpkgs, disko, ... }: {
    nixosConfigurations = {
      # <nixmgr:hosts>
    };
  };
}
`
	if err := os.WriteFile(flakePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := AddFlakeOutput(flakePath, "zeus"); err != nil {
		t.Fatalf("AddFlakeOutput(zeus): %v", err)
	}

	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, `zeus = nixpkgs.lib.nixosSystem`) {
		t.Errorf("expected zeus entry, got:\n%s", content)
	}
	if !strings.Contains(content, "# <nixmgr:hosts>") {
		t.Errorf("marker should be preserved, got:\n%s", content)
	}

	// Add a second host
	if err := AddFlakeOutput(flakePath, "athena"); err != nil {
		t.Fatalf("AddFlakeOutput(athena): %v", err)
	}

	data, err = os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}

	content = string(data)
	if !strings.Contains(content, `zeus = nixpkgs.lib.nixosSystem`) {
		t.Errorf("zeus should still be present, got:\n%s", content)
	}
	if !strings.Contains(content, `athena = nixpkgs.lib.nixosSystem`) {
		t.Errorf("athena should be present, got:\n%s", content)
	}
	if !strings.Contains(content, "# <nixmgr:hosts>") {
		t.Errorf("marker should still be preserved, got:\n%s", content)
	}

	t.Logf("Final flake.nix:\n%s", content)
}
