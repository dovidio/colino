#!/bin/bash

set -e

echo "🎬 Starting Colino demo generation in clean environment..."
echo "=================================================="
echo "🏠 Current HOME: $HOME"
echo "🗂️  Expected config: $HOME/.config/colino/config.yaml"

# Backup existing golden file if it exists
if [ -f "demo/golden.ascii" ]; then
    echo "📦 Backing up existing golden file..."
    cp demo/golden.ascii demo/golden.ascii.tmp
    echo "✅ Golden file backed up"
else
    echo "ℹ️ No existing golden file found - this might be the first run"
fi

echo ""
echo "🔧 Building Colino and demo server..."

# Build colino
echo "Building colino..."
go build -o colino ./cmd/colino

# Build demo server
echo "Building demo server..."
go build -o demo-server ./cmd/demo-server

echo ""
echo "🚀 Starting demo server on port 8080..."
./demo-server -port 8080 &
DEMO_SERVER_PID=$!

# Give the server time to start
echo "⏳ Waiting for server to start..."
sleep 2

# Verify server is running
if ! kill -0 $DEMO_SERVER_PID 2>/dev/null; then
    echo "❌ Demo server failed to start"
    exit 1
fi

echo "✅ Demo server is running (PID: $DEMO_SERVER_PID)"

echo ""
echo "🎥 Recording VHS demo..."
ls
cp ./colino demo/colino
vhs demo/demo.tape

# Clean up demo server
echo ""
echo "🧹 Stopping demo server..."
kill $DEMO_SERVER_PID 2>/dev/null || true
wait $DEMO_SERVER_PID 2>/dev/null || true

echo ""
echo "🔍 Validating generated files..."

# Check if demo files were generated
if [ ! -f "demo/demo.gif" ]; then
    echo "❌ Demo GIF not generated"
    exit 1
fi

if [ ! -f "demo/golden.ascii" ]; then
    echo "❌ Golden ASCII file not generated"
    exit 1
fi

echo "✅ Demo files generated successfully:"
echo "   📹 demo/demo.gif ($(stat -f%z demo/demo.gif 2>/dev/null || stat -c%s demo/demo.gif) bytes)"
echo "   📄 demo/golden.ascii ($(stat -f%z demo/golden.ascii 2>/dev/null || stat -c%s demo/golden.ascii) bytes)"

echo ""
echo "🔎 Comparing with previous version..."

if [ -f "demo/golden.ascii.tmp" ]; then
    if ! diff -u demo/golden.ascii.tmp demo/golden.ascii; then
        echo ""
        echo "❌ Demo output has changed!"
        echo ""
        echo "The generated demo differs from the committed snapshot."
        echo "This may be due to intended changes or unexpected behavior."
        echo ""
        echo "Next steps:"
        echo "1. Review the changes above"
        echo "2. Check demo/demo.gif visually if needed"
        echo "3. If changes are correct, commit the updated files:"
        echo "   git add demo/demo.gif demo/golden.ascii"
        echo "   git commit -m 'Update demo snapshots'"
        echo ""
        echo "If changes are unexpected, investigate the cause."
        exit 1
    else
        echo "✅ Demo output matches expected snapshots"
        # Clean up temporary file
        rm -f demo/golden.ascii.tmp
    fi
else
    echo ""
    echo "⚠️ No previous golden file found for comparison"
    echo "This might be the first time the demo is being generated."
    echo "If this looks correct, commit the generated files:"
    echo "  git add demo/demo.gif demo/golden.ascii"
    echo "  git commit -m 'Add initial demo snapshots'"
    exit 1
fi

echo ""
echo "🎉 Demo generation completed successfully!"
echo "=================================================="
