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

      go = pkgs.go_1_26.overrideAttrs (finalAttrs: _: {
        version = "1.26.11";
        src = pkgs.fetchurl {
          url = "https://go.dev/dl/go${finalAttrs.version}.src.tar.gz";
          hash = "sha256-1EEBsP8n7SpMda1GwXucdDWSCeLrLKfJ32y0/baOGWk=";
        };
      });

    in {
      devShell = pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          golangci-lint
          moq

          gh
          goreleaser
        ]
        ++ (if pkgs.stdenv.isDarwin then [ apple-sdk_15 ] else []);
      };
    });
}
