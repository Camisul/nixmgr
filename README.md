# nixmgr

A CLI tool for managing a fleet of NixOS machines using flakes. It automates the
boring parts of adding a new host: picking a name, pointing DNS at it via
Cloudflare, scaffolding a NixOS configuration, and wiring it into your flake.

## How it works

```
nixmgr add 203.0.113.10
```

This single command:

1. Picks the next available name from the pool defined in `nixmgr.toml`
2. Creates a Cloudflare DNS A record (`<name>.<domain>` -> the IP)
3. Scaffolds `hosts/<name>/configuration.nix`
4. Adds a `nixosConfigurations.<name>` entry to `flake.nix`

You then customize the host config and deploy with
[nixos-anywhere](https://github.com/nix-community/nixos-anywhere).

## Setup

### Prerequisites

- Go 1.21+
- A Cloudflare account with an API token that has DNS edit permissions
- The zone ID for your domain

### Build

```sh
go build -o nixmgr .
```

### Configuration

Edit `nixmgr.toml` in the project root:

```toml
# The domain under which host subdomains are created.
domain = "example.com"

# Pool of host names, assigned in order. First unused name wins.
names = [
  "zeus",
  "athena",
  "hermes",
  "apollo",
  "artemis",
]
```

Set the required environment variables:

```sh
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"
```

## Usage

### Add a host

```sh
nixmgr add 203.0.113.10
```

```
Selected name: zeus
FQDN:          zeus.example.com
IP:             203.0.113.10

Creating DNS record zeus.example.com -> 203.0.113.10 ... done
Scaffolding hosts/zeus/configuration.nix ... done
Adding flake output nixosConfigurations.zeus ... done

Host 'zeus' added successfully.

Next steps:
  1. Edit hosts/zeus/configuration.nix to customize the host
  2. Deploy with: nixos-anywhere --flake .#zeus root@203.0.113.10
```

### Dry run

Preview what would happen without making any changes:

```sh
nixmgr add --dry-run 203.0.113.10
```

### Add and deploy immediately

Run `nixos-anywhere` automatically after scaffolding and flake updates:

```sh
nixmgr add --run 203.0.113.10
```

## Nix modules

### `modules/base.nix`

Shared configuration applied to every host:

- GRUB bootloader with EFI support
- OpenSSH enabled
- Base packages: `curl`, `gitMinimal`, `vim`
- Root SSH authorized keys (edit to add your own)

### `modules/disk-config.nix`

Disko disk layout applied to every host:

- GPT partition table on `/dev/sda` (overridable with `lib.mkForce`)
- 1M BIOS boot partition
- 500M EFI system partition mounted at `/boot`
- Remaining space as LVM PV -> VG `pool` -> LV `root` (ext4, mounted at `/`)

### Per-host config

Each scaffolded `hosts/<name>/configuration.nix` imports both shared modules
and sets the hostname. Add host-specific configuration there.

## Deploying

After adding a host, deploy it with nixos-anywhere:

```sh
nixos-anywhere --flake .#zeus root@203.0.113.10
```

For hosts with a different disk device, override it in the host config:

```nix
disko.devices.disk.main.device = lib.mkForce "/dev/vda";
```

## Tests

```sh
go test ./...
```
