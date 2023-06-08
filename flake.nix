{
  description = "Multi-purpose Telegram bot";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs, ... }:

    let
      forAllSystems = function:
        nixpkgs.lib.genAttrs [
          "x86_64-linux"
          "aarch64-linux"
          "x86_64-darwin"
          "aarch64-darwin"
        ]
          (system: function nixpkgs.legacyPackages.${system});

      version =
        if (self ? shortRev)
        then self.shortRev
        else "dev";
    in
    {

      devShells = forAllSystems
        (pkgs: {
          default = pkgs.mkShell {
            packages = [
              pkgs.go
              pkgs.golangci-lint
            ];
            shellHook = ''
              export PRINT_MSGS=1
              export PRETTY_PRINT_LOG=1
              export DEBUG=1
            '';
          };
        });


      packages = forAllSystems
        (pkgs: {
          default =
            pkgs.buildGoModule
              {
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
    };
}
