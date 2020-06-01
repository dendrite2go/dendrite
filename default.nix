{ buildGoModule
, nix-gitignore
}:

buildGoModule {
  pname = "dendrite";
  version = "0.0.1";
  src = nix-gitignore.gitignoreSource [] ./.;
  goPackagePath = "github.com/dendrite2go/dendrite";
  goDeps = ./deps.nix;
  modSha256 = "1r332237s6lx34n53d8q136hh6x6nava4cazqy018842aphridx2";
}
