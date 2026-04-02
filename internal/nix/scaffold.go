package nix

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed tmpl/flake.nix.gotmpl
var flakeTempl string
var flakeTemplate = template.Must(template.New("flake").Parse(flakeTempl))

//go:embed tmpl/config.toml.gotmpl
var configTempl string
var configTemplate = template.Must(template.New("config-toml").Parse(configTempl))

//go:embed tmpl/configuration.nix.gotmpl
var hostTempl string
var hostTemplate = template.Must(template.New("config").Parse(hostTempl))

//go:embed tmpl/disk-config.nix.gotmpl
var diskTempl string
var diskTemplate = template.Must(template.New("config").Parse(diskTempl))

// InitProject scaffolds a new nixmgr project in the given directory.
// It creates flake.nix, nixmgr.toml, and an empty hosts/ directory.
func InitProject(dir, domain string) error {
	// Write flake.nix
	flakePath := filepath.Join(dir, "flake.nix")
	f, err := os.Create(flakePath)
	defer f.Close()

	if err != nil {
		return fmt.Errorf("creating flake.nix: %w", err)
	}

	if err := flakeTemplate.Execute(f, nil); err != nil {
		return fmt.Errorf("writing flake.nix: %w", err)
	}

	// Write nixmgr.toml
	tomlPath := filepath.Join(dir, "nixmgr.toml")
	t, err := os.Create(tomlPath)
	defer t.Close()
	if err != nil {
		return fmt.Errorf("creating nixmgr.toml: %w", err)
	}
	if err := configTemplate.Execute(t, struct{ Domain string }{Domain: domain}); err != nil {
		return fmt.Errorf("writing nixmgr.toml: %w", err)
	}

	// Create hosts/ directory with .gitkeep
	hostsDir := filepath.Join(dir, "hosts")
	if err := os.MkdirAll(hostsDir, 0o755); err != nil {
		return fmt.Errorf("creating hosts dir: %w", err)
	}
	gitkeep := filepath.Join(hostsDir, ".gitkeep")
	if err := os.WriteFile(gitkeep, []byte{}, 0o644); err != nil {
		return fmt.Errorf("creating hosts/.gitkeep: %w", err)
	}

	return nil
}

func ScaffoldHost(hostsDir, name string, sopsEnabled bool) error {
	hostDir := filepath.Join(hostsDir, name)
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		return fmt.Errorf("creating host dir: %w", err)
	}

	{
		configPath := filepath.Join(hostDir, "configuration.nix")
		f, err := os.Create(configPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", configPath, err)
		}
		defer f.Close()

		if err := hostTemplate.Execute(f, struct {
			Name        string
			SopsEnabled bool
		}{Name: name, SopsEnabled: sopsEnabled}); err != nil {
			return fmt.Errorf("writing template: %w", err)
		}
	}
	{
		discConfigPath := filepath.Join(hostDir, "disc-config.nix")

		f, err := os.Create(discConfigPath)
		if err != nil {
			return fmt.Errorf("creating %s: %w", discConfigPath, err)
		}
		defer f.Close()

		if err := diskTemplate.Execute(f, struct{ Name string }{Name: name}); err != nil {
			return fmt.Errorf("writing template: %w", err)
		}
	}
	return nil
}

// AddFlakeOutput inserts a nixosConfigurations entry into flake.nix at the marker comment.
func AddFlakeOutput(flakePath, name string, sopsEnabled bool) error {
	data, err := os.ReadFile(flakePath)
	if err != nil {
		return fmt.Errorf("reading flake.nix: %w", err)
	}

	content := string(data)
	if sopsEnabled {
		content = ensureSopsNixInput(content)
	}

	marker := "# <nixmgr:hosts>"
	modules := `disko.nixosModules.disko
          ./hosts/%s/configuration.nix`
	if sopsEnabled {
		modules = `disko.nixosModules.disko
          sops-nix.nixosModules.sops
          ./hosts/%s/configuration.nix`
	}

	entry := fmt.Sprintf(`%s = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          %s
        ];
      };

      `, name, fmt.Sprintf(modules, name))

	if !strings.Contains(content, marker) {
		return fmt.Errorf("marker %q not found in flake.nix — cannot inject host", marker)
	}

	newContent := strings.Replace(content, marker, entry+marker, 1)

	if err := os.WriteFile(flakePath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("writing flake.nix: %w", err)
	}

	return nil
}

func UpdateSopsMachineKeys(sopsPath, name, ageRecipient string) error {
	const marker = "# <nixmgr:machine-keys>"
	ageRecipient = strings.TrimSpace(ageRecipient)
	if ageRecipient == "" {
		return fmt.Errorf("empty age recipient for host %q", name)
	}

	entry := fmt.Sprintf("          - %s # nixmgr:%s\n", ageRecipient, name)

	data, err := os.ReadFile(sopsPath)
	if os.IsNotExist(err) {
		initial := fmt.Sprintf(`keys:
  - &admin age1REPLACE_WITH_ADMIN_KEY

creation_rules:
  - path_regex: secrets/[^/]+\.(yaml|json|env|ini)$
    key_groups:
      - age:
          - *admin
          %s
%s`, marker, entry)
		if err := os.WriteFile(sopsPath, []byte(initial), 0o644); err != nil {
			return fmt.Errorf("writing .sops.yaml: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading .sops.yaml: %w", err)
	}

	content := string(data)
	if strings.Contains(content, "# nixmgr:"+name) {
		return nil
	}

	if !strings.Contains(content, marker) {
		return fmt.Errorf(".sops.yaml missing %q marker; add it under your age recipient list to allow nixmgr to append machine keys", marker)
	}

	updated := strings.Replace(content, marker, entry+marker, 1)
	if err := os.WriteFile(sopsPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("writing .sops.yaml: %w", err)
	}

	return nil
}

func ensureSopsNixInput(content string) string {
	if !strings.Contains(content, "sops-nix") {
		content = strings.Replace(content, "inputs = {", `inputs = {
    sops-nix = {
      url = "github:Mic92/sops-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };`, 1)

		content = strings.Replace(content, "outputs = { self, nixpkgs, disko, ... }:", "outputs = { self, nixpkgs, disko, sops-nix, ... }:", 1)
	}

	return content
}
