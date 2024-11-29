{
  description = "Welcome to the Purple Clay Daggerverse! Dagger functions to streamline your CI";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";

    dagger = {
        # Locked to version 0.14.0
        url = "github:dagger/nix/9852fdddcdcb52841275ffb6a39fa1524d538d5a";
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
            shellcheck
          ];
        };
      }
    );
}
