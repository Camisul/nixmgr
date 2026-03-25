{ config, lib, pkgs, ... }:

{
  imports = [
    <nixpkgs/nixos/modules/installer/scan/not-detected.nix>
    <nixpkgs/nixos/modules/profiles/qemu-guest.nix>
  ];

  boot.loader.grub = {
    efiSupport = true;
    efiInstallAsRemovable = true;
  };

  services.openssh.enable = true;

  environment.systemPackages = with pkgs; [
    curl
    gitMinimal
    vim
  ];

  users.users.root.openssh.authorizedKeys.keys = [
    # Add your SSH public key(s) here
    # "ssh-ed25519 AAAA..."
  ];

  system.stateVersion = "24.11";
}
