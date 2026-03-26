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

func TestInitProject(t *testing.T) {
	dir := t.TempDir()

	if err := InitProject(dir, "infra.example.com"); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Check flake.nix
	flakeData, err := os.ReadFile(filepath.Join(dir, "flake.nix"))
	if err != nil {
		t.Fatalf("reading flake.nix: %v", err)
	}
	flake := string(flakeData)
	if !strings.Contains(flake, "# <nixmgr:hosts>") {
		t.Errorf("flake.nix should contain hosts marker, got:\n%s", flake)
	}
	if !strings.Contains(flake, "github:nix-community/disko") {
		t.Errorf("flake.nix should contain disko input, got:\n%s", flake)
	}
	if !strings.Contains(flake, "github:NixOS/nixpkgs/nixpkgs-unstable") {
		t.Errorf("flake.nix should contain nixpkgs input, got:\n%s", flake)
	}

	// Check nixmgr.toml
	tomlData, err := os.ReadFile(filepath.Join(dir, "nixmgr.toml"))
	if err != nil {
		t.Fatalf("reading nixmgr.toml: %v", err)
	}
	tomlContent := string(tomlData)
	if !strings.Contains(tomlContent, `domain = "infra.example.com"`) {
		t.Errorf("nixmgr.toml should contain domain, got:\n%s", tomlContent)
	}
	if !strings.Contains(tomlContent, `"zeus"`) {
		t.Errorf("nixmgr.toml should contain default names, got:\n%s", tomlContent)
	}

	// Check hosts/.gitkeep
	if _, err := os.Stat(filepath.Join(dir, "hosts", ".gitkeep")); err != nil {
		t.Errorf("hosts/.gitkeep should exist: %v", err)
	}
}

func TestInitProjectAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()

	// First init should succeed
	if err := InitProject(dir, "example.com"); err != nil {
		t.Fatalf("first InitProject: %v", err)
	}

	// Second init should still succeed (InitProject itself doesn't check;
	// the command layer handles the "already initialized" check)
	// This test just verifies files are overwritten without error
	if err := InitProject(dir, "other.com"); err != nil {
		t.Fatalf("second InitProject: %v", err)
	}

	tomlData, err := os.ReadFile(filepath.Join(dir, "nixmgr.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlData), `domain = "other.com"`) {
		t.Errorf("should have updated domain, got:\n%s", string(tomlData))
	}
}

func TestInitProjectFlakeCompatibleWithAddFlakeOutput(t *testing.T) {
	dir := t.TempDir()

	if err := InitProject(dir, "example.com"); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// The generated flake.nix should be compatible with AddFlakeOutput
	flakePath := filepath.Join(dir, "flake.nix")
	if err := AddFlakeOutput(flakePath, "zeus"); err != nil {
		t.Fatalf("AddFlakeOutput after init: %v", err)
	}

	data, err := os.ReadFile(flakePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `zeus = nixpkgs.lib.nixosSystem`) {
		t.Errorf("should contain zeus entry, got:\n%s", content)
	}
	if !strings.Contains(content, "# <nixmgr:hosts>") {
		t.Errorf("marker should be preserved, got:\n%s", content)
	}
}
