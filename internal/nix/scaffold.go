package nix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var hostTemplate = template.Must(template.New("config").Parse(`{ config, lib, pkgs, ... }:

{
  imports = [
    ../../modules/base.nix
    ../../modules/disk-config.nix
  ];

  networking.hostName = "{{.Name}}";

  # Add host-specific configuration below
}
`))

func ScaffoldHost(hostsDir, name string) error {
	hostDir := filepath.Join(hostsDir, name)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return fmt.Errorf("creating host dir: %w", err)
	}

	configPath := filepath.Join(hostDir, "configuration.nix")

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", configPath, err)
	}
	defer f.Close()

	if err := hostTemplate.Execute(f, struct{ Name string }{Name: name}); err != nil {
		return fmt.Errorf("writing template: %w", err)
	}

	return nil
}

// AddFlakeOutput inserts a nixosConfigurations entry into flake.nix at the marker comment.
func AddFlakeOutput(flakePath, name string) error {
	data, err := os.ReadFile(flakePath)
	if err != nil {
		return fmt.Errorf("reading flake.nix: %w", err)
	}

	content := string(data)

	marker := "# <nixmgr:hosts>"
	entry := fmt.Sprintf(`%s = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          disko.nixosModules.disko
          ./hosts/%s/configuration.nix
        ];
      };

      `, name, name)

	if !strings.Contains(content, marker) {
		return fmt.Errorf("marker %q not found in flake.nix — cannot inject host", marker)
	}

	newContent := strings.Replace(content, marker, entry+marker, 1)

	if err := os.WriteFile(flakePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("writing flake.nix: %w", err)
	}

	return nil
}
