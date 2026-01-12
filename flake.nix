{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-compat = {
      url = "github:edolstra/flake-compat";
      flake = false;
    };
  };

  outputs = {
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {
        inherit system;
        config.allowUnfree = true;
      };

    in {
      devShell = pkgs.mkShell {
        buildInputs = with pkgs; [
          go_1_25
          golangci-lint
          moq
        ]
        ++ (if pkgs.stdenv.isDarwin then [ apple-sdk_15 ] else []);
      };
    });
}
