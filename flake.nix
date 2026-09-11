{
  description = "Dev shell for CGO cross-compilation (386, armv7, arm64)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
      ];

      forAllSystems = f:
        builtins.listToAttrs (map (system: {
          name = system;
          value = f system;
        }) systems);
    in {
      devShells = forAllSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in {
          default = pkgs.mkShell {
            buildInputs = [
              pkgs.bash
              pkgs.go_1_27
              pkgs.golangci-lint
              pkgs.patchelf
              pkgs.revive
              pkgs.gnumake

              pkgs.gcc
              pkgs.pkg-config
              pkgs.openssl
              pkgs.tcl
            ];

            shellHook = ''
              unset GOROOT

              export OPENSSL_CURRENT_INCLUDE=${pkgs.openssl.dev}/include
              export OPENSSL_CURRENT_LIB=${pkgs.openssl.out}/lib

              export SQLCIPHER_TCLSH=${pkgs.tcl}/bin/tclsh
              export SQLCIPHER_TCL_CONFIG_DIR=${pkgs.tcl}/lib

              echo "Using go config:"
              echo "  GOROOT   = $(go env GOROOT)"
              echo "  GOCACHE  = $(go env GOCACHE)"
              echo "  GOPATH   = $(go env GOPATH)"
            '';
          };
        }
      );
    };
}
