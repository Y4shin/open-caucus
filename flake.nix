{
  description = "plai-api development environment with Go, OpenAPI tooling, and code generation";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        go = pkgs.go_1_25;

        # Build mcp-gopls from source
        mcp-gopls = pkgs.buildGoModule rec {
          pname = "mcp-gopls";
          version = "2.0.1";

          src = pkgs.fetchFromGitHub {
            owner = "hloiseau";
            repo = "mcp-gopls";
            rev = "v${version}";
            hash = "sha256-ulv6XBD5evNzNfHiPigKW3Ghreia/QZnicaopJmzNKM="; # Update this hash
          };

          vendorHash = "sha256-W8hlCGf4QdFbKc3QFc9pa4MWBhnp5A5GvWFNzg0BEhw="; # Update this hash

          subPackages = [ "cmd/mcp-gopls" ];

          meta = with pkgs.lib; {
            description = "Model Context Protocol server for Go using gopls";
            homepage = "https://github.com/hloiseau/mcp-gopls";
            license = licenses.asl20;
            maintainers = [ ];
          };
        };

        # Build mcp-taskfile-server from source
        mcp-taskfile-server = pkgs.buildGoModule rec {
          pname = "mcp-taskfile-server";
          version = "main";

          src = pkgs.fetchFromGitHub {
            owner = "rsclarke";
            repo = "mcp-taskfile-server";
            rev = "${version}";
            hash = "sha256-tCfjlyabjWO5VYhWCIypD84tC65TiJP6vGD/oL+7/+s="; # Will be replaced with actual hash on first build
          };

          vendorHash = "sha256-c7aWabtrj4sqPExoQS9xVeB2whXvS3iD9Dg3yHd2NGE="; # Will be replaced with actual hash on first build

          meta = with pkgs.lib; {
            description = "Model Context Protocol server for Taskfile";
            homepage = "https://github.com/rsclarke/mcp-taskfile-server";
            license = licenses.mit;
            maintainers = [ ];
          };
        };

      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go
            gopls
            gotools
            golangci-lint

            # Task runner
            go-task

            # MCP servers for AI assistance
            mcp-gopls
            mcp-taskfile-server
          ];

          shellHook = ''
            echo "🚀 conference-tool development environment"
            echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
            echo ""
            echo "Available tools:"
            echo "  • Go $(go version | cut -d' ' -f3)"
            echo "  • Task (Taskfile runner)"
            echo "  • mcp-gopls (AI MCP server for Go)"
            echo "  • mcp-taskfile-server (AI MCP server for Taskfile)"
            echo ""
            echo "Quick start:"
            echo "  task run         # Run locally"
            echo ""
          '';
        };
      }
    );
}
