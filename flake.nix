{
  description = "uestc qsh telecom login script";

  inputs = {
    flake-parts.url = "github:hercules-ci/flake-parts";
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = inputs: inputs.flake-parts.lib.mkFlake { inherit inputs; } {
    systems = [ "x86_64-linux" "aarch64-linux" ];
    perSystem = { pkgs, ... }: {
      packages = rec {
        qsh-telecom-autologin = with pkgs; buildGoModule rec {
          name = "qsh-telecom-autologin";

          src = lib.cleanSource ./.;

          vendorHash = null;

          ldflags = [ "-s" "-w" ];
        };
        default = qsh-telecom-autologin;
      };
    };
  };
}
