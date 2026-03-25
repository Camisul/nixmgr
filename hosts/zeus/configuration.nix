{ config, lib, pkgs, ... }:

{
  imports = [
    ../../modules/base.nix
    ../../modules/disk-config.nix
  ];

  networking.hostName = "zeus";

  # Add host-specific configuration below
}
