{
  description = "Welcome to the Daggerverse! Dagger functions to streamline your CI";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    dagger = {
        # Locked to version 0.12.5 for the time being
        url = "github:dagger/nix/5053689af7d18e67254ba0b2d60fa916b7370104";
        inputs = {
            nixpkgs.follows = "nixpkgs";
        };
    };
  };

  outputs = { self, nixpkgs, flake-utils, dagger }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      with pkgs;
      {
        devShells.default = mkShell {
          buildInputs = [
            dagger.packages.${system}.dagger
            git
            go
            gofumpt
            nixd
          ];
        };
      }
    );
}
