{
  description = "Multi-purpose Telegram bot";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils, ... }:

    let
      version =
        if (self ? shortRev)
        then self.shortRev
        else "dev";
    in

    utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go ];
          shellHook = ''
            export PRINT_MSGS=1
            export PRETTY_PRINT_LOG=1
            export DEBUG=1
          '';
        };
        packages.default = pkgs.buildGoModule {
          pname = "gobot";
          inherit version;
          src = pkgs.lib.cleanSource self;

          # Update the hash if go dependencies change!
          # vendorHash = pkgs.lib.fakeHash;
          vendorHash = "sha256-j8jigWSW/7459j4NTeCrDPH0QNj7ZDUVZI8wR8xC+UY=";

          ldflags = [ "-s" "-w" ];

          meta = {
            description = "Multi-purpose Telegram bot";
            homepage = "https://github.com/Brawl345/gobot";
            license = pkgs.lib.licenses.unlicense;
            platforms = pkgs.lib.platforms.darwin ++ pkgs.lib.platforms.linux;
          };
        };
      });
}
