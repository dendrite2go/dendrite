{ buildGoModule
, nix-gitignore
}:

buildGoModule {
  pname = "dendrite";
  version = "0.0.1";
  src = nix-gitignore.gitignoreSource [] ./.;
  goPackagePath = "github.com/dendrite2go/dendrite";
  goDeps = ./deps.nix;
  modSha256 = "0igd5rvdzjbqihf1bdll5kch3b5z16gzm21jhzsnwb2n6hwywlf8";
}
