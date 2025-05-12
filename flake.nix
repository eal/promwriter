{
  description = "promwriter";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
      v = "v0.0.1";
      

      promwriter = pkgs.buildGoModule rec {
        pname = "promwriter";
        version = v;
        vendorHash = "sha256-5QjjI/mr/yBDGHcVJdG6lt6kATdkvRXRU/KpBobM3Qc=";
        src = self;
      };

      dockerImage = pkgs.dockerTools.buildImage {
        name = "promwriter";
        tag = promwriter.version;
        contents = [ promwriter ];
        config = {
          Cmd = [ "/bin/promwriter" ];
          # Env = [ "REMOTE_WRITE_URL=http://localhost:9090/api/v1/write" ];
        };
      };

    in
    {
      defaultPackage.${system} = promwriter;
      packages.${system}.dockerImage = dockerImage;

      
      defaultDevShell.${system} = pkgs.mkShell {
        buildInputs = [ promwriter ];
        shellHook = ''
          export VERSION=${promwriter.version}
        '';
      };
    };
}
