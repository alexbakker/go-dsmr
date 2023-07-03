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
          vendorSha256 = "sha256-7cc7ORKcUJC9i9t0b+WL36S9/geeOCYcm/lMbjr7nKQ=";
        };
      };
      devShells.default = with pkgs; mkShell {
        buildInputs = [
          go
        ];
      };
    }
  );
}
