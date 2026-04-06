{
  description = "";
  inputs = {
    nixpkgs.url = "github:NixOs/nixpkgs/nixos-unstable";
  };

  outputs =
    {
      self,
      nixpkgs,
      ...
    }:
    let
      supportedSystems = [
        "x86_64-linux"
        "x86_64-darwin"
        "aarch64-linux"
        "aarch64-darwin"
      ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      pkgsFor = system: nixpkgs.legacyPackages.${system};
      pname = "nixmgr";
      owner = "camisul";
      version = "0.1.0";
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.buildGoModule {
            inherit pname;
            inherit version; 
            src = ./.;
            vendorHash = "sha256-9woA8qrKRVWIpN2jRnhXmT3vnJOvis0d5nWHyhWqPZE=";
          };
        }
      );

      # Development shell
      devShells = forAllSystems (
        system:
        let
          pkgs = pkgsFor system;
        in
        {
          default = pkgs.mkShell {
            buildInputs = [
              pkgs.go
              pkgs.gopls
              pkgs.nixfmt-rfc-style
            ];
            # Add a Git pre-commit hook.
            # shellHook = onchg.shellHook.${system};
          };
          ci = pkgs.mkShell {
            # We already have Go installed.
            buildInputs = [ pkgs.nixfmt-rfc-style ];
          };
        }
      );
    };
}
