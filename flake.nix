{
  description = "Dev environment for chirpy";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = {
    self,
    nixpkgs,
  }: {
    devShells.x86_64-linux.default = let
      pkgs = import nixpkgs {system = "x86_64-linux";};
    in
      pkgs.mkShell {
        buildInputs = with pkgs; [
          go
          zsh
          openssl
          unixtools.xxd
          sqlite
          ffmpeg_6
        ];

        shellHook = ''
          export GOPATH=$PWD/.gopath
          export GOBIN=$GOPATH/bin
          export PATH=$GOBIN:$PATH
          mkdir -p "$GOBIN"
          go mod tidy
          if [ -z "$ZSH_VERSION" ]; then
            exec zsh
          fi
        '';
      };
  };
}
