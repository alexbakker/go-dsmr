{
  description = "Nix flake for go-dsmr";

  inputs = {
    flake-utils.url = "github:numtide/flake-utils";
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, flake-utils, nixpkgs }:
  flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = nixpkgs.legacyPackages.${system};
    in rec {
      packages = flake-utils.lib.flattenTree rec {
        default = dsmr-exporter;
        dsmr-exporter = with pkgs; buildGoModule rec {
          name = "dsmr-exporter";
          src = ./.;

          subPackages = [ "cmd/dsmr-exporter" ];
          vendorHash = "sha256-7cc7ORKcUJC9i9t0b+WL36S9/geeOCYcm/lMbjr7nKQ=";
        };
      };
      devShells.default = with pkgs; mkShell {
        buildInputs = [
          go
        ];
      };
      nixosModules.default = ({ pkgs, ... }: {
        imports = [ ./module.nix ];
        nixpkgs.overlays = [
          (_self: _super: {
            dsmr-exporter = self.packages.${pkgs.system}.dsmr-exporter;
          })
        ];
      });
    }
  );
}
