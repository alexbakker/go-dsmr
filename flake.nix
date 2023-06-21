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
          src = ./cmd/dsmr-exporter;

          vendorSha256 = lib.fakeSha256;
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
