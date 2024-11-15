{ pkgs ? import <nixpkgs> { } }:

let
  eval = pkgs.lib.evalModules {
    modules = [
      ./module.nix
            {
        _module.check = false;
      }
    ];
  };
in
pkgs.nixosOptionsDoc {
  options = eval.options;
  transformOptions = opt: opt;
}
