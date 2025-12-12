#!/bin/bash

echo "🔧 Setting up Node.js GraphQL API..."
echo ""

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+ first."
    exit 1
fi

echo "✓ Node.js version: $(node --version)"

# Check if wasm_exec.js exists
if [ ! -f "wasm_exec.js" ]; then
    echo "📦 Copying wasm_exec.js..."
    cp ../../wasm/wasm_exec.js .
    echo "✓ wasm_exec.js copied"
fi

# Install dependencies
echo ""
echo "📦 Installing dependencies..."
npm install

# Check if SyndrDB is running
echo ""
echo "🔍 Checking SyndrDB connection..."
if nc -z 127.0.0.1 1776 2>/dev/null; then
    echo "✓ SyndrDB is running on port 1776"
else
    echo "⚠️  SyndrDB does not appear to be running on port 1776"
    echo "   Please start your SyndrDB server before running the API"
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Ensure SyndrDB server is running"
echo "  2. Create the todos bundle (see README.md)"
echo "  3. Run: npm start"
echo "  4. Open: http://localhost:3000"
echo ""
