#!/bin/bash
# setup_rift.sh - Setup script for Rift Editor on Ubuntu
# Usage: ./setup_rift.sh [install_path]

INSTALL_PATH="${1:-$HOME/.local/bin}"
RIFT_DIR="${2:-$HOME/.rift}"

echo "=== Rift Editor Setup ==="
echo "Install path: $INSTALL_PATH"
echo "Rift directory: $RIFT_DIR"
echo ""

# Create directories
mkdir -p "$INSTALL_PATH"
mkdir -p "$RIFT_DIR"

# Check if rift binary exists in current directory
if [ -f "rift" ]; then
    echo "Installing rift binary..."
    cp rift "$INSTALL_PATH/"
    chmod +x "$INSTALL_PATH/rift"
else
    echo "Building rift from source..."
    if command -v go &> /dev/null; then
        go build -o "$INSTALL_PATH/rift" ./cmd/rift
        chmod +x "$INSTALL_PATH/rift"
    else
        echo "ERROR: Go not found and no pre-built binary available"
        exit 1
    fi
fi

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_PATH:"* ]]; then
    echo "Adding $INSTALL_PATH to PATH..."
    echo "export PATH=\"$INSTALL_PATH:\$PATH\"" >> "$HOME/.bashrc"
    echo "Please run: source ~/.bashrc"
fi

# Create convenience function for rift with cd
cat > "$RIFT_DIR/rift_func.sh" << 'EOF'
#!/bin/bash
# Rift editor wrapper function - passes full paths to binary
rift() {
    if [ $# -eq 0 ]; then
        # No args - open rift in current directory
        command rift
    elif [ -d "$1" ]; then
        # Directory argument - pass full path
        command rift "$(cd "$1" && pwd)"
    elif [ -f "$1" ]; then
        # File argument - pass full path
        command rift "$(readlink -f "$1")"
    else
        # Unknown - let rift handle it
        command rift "$@"
    fi
}
EOF

# Add function to bashrc if not already there
if ! grep -q "source $RIFT_DIR/rift_func.sh" "$HOME/.bashrc"; then
    echo "Adding rift function to .bashrc..."
    echo "" >> "$HOME/.bashrc"
    echo "# Rift Editor function" >> "$HOME/.bashrc"
    echo "source $RIFT_DIR/rift_func.sh" >> "$HOME/.bashrc"
fi

echo ""
echo "=== Setup Complete ==="
echo "Binary installed to: $INSTALL_PATH/rift"
echo "Run 'source ~/.bashrc' to activate the rift function"
echo ""
echo "Usage examples:"
echo "  rift                    # Open rift in current directory"
echo "  rift /path/to/folder    # Open rift in specific folder"
echo "  rift /path/to/file.go   # Open specific file in rift"
