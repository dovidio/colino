{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    # Core system utilities needed for --pure environment
    coreutils
    bash
    gnused
    gnugrep
    findutils
    which

    # Go toolchain
    go
    gopls

    # VHS and dependencies
    vhs
    ffmpeg
    ttyd

    # Additional tools
    git
  ];

  # Set environment variables for demo
  shellHook = ''
    echo "🎬 Colino Demo Environment"
    echo "=========================="
    echo "Go version: $(go version)"
    echo "VHS version: $(vhs version)"
    echo "ffmpeg version: $(ffmpeg -version | head -n1)"
    echo "=========================="
    echo ""

    # Ensure PATH includes all necessary binaries for VHS
    export PATH=${pkgs.coreutils}/bin:${pkgs.bash}/bin:${pkgs.gnused}/bin:${pkgs.gnugrep}/bin:${pkgs.findutils}/bin:${pkgs.which}/bin:${pkgs.go}/bin:${pkgs.vhs}/bin:${pkgs.ffmpeg}/bin:$PATH

    # Set up Go caching for faster builds
    export GOCACHE=$(pwd)/.go-cache
    export GOMODCACHE=$(pwd)/.go-mod-cache
    export GO_CACHE=$(pwd)/.go-cache

    # Create cache directories if they don't exist
    mkdir -p $GOCACHE $GOMODCACHE

    echo "🚀 Go cache configured:"
    echo "  GOCACHE: $GOCACHE"
    echo "  GOMODCACHE: $GOMODCACHE"
    echo ""
    echo "Run 'make demo' to generate demo files in clean environment"
  '';
}