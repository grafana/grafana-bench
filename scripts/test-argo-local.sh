#!/bin/bash
set -e

# Local script to test Argo workflow validation without CI
echo "🚀 Testing Argo workflow validation locally..."

# Check prerequisites
if ! command -v jsonnet &> /dev/null; then
    echo "❌ jsonnet not found. Run: make install-deps"
    exit 1
fi

if ! command -v argo &> /dev/null; then
    echo "📦 Installing Argo Workflows CLI..."
    ARGO_VERSION="v3.7.3"
    curl -sLO "https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-linux-amd64.gz"
    gunzip argo-linux-amd64.gz
    chmod +x argo-linux-amd64
    sudo mv argo-linux-amd64 /usr/local/bin/argo
    echo "✅ Argo CLI installed: $(argo version --short)"
fi

# Clean up previous runs
rm -rf test-output/
mkdir -p test-output/

# Generate libsonnet with test version
echo "🔧 Generating libsonnet..."
cd generators/libsonnet
go run main.go -o ../../test-output -version v1.0.0-test

# Copy test dependencies and fix import paths
echo "📋 Copying dependencies and fixing import paths..."
cp -r deps ../../test-output/
cd ../..

# Rewrite import paths in the generated libsonnet
sed -i.bak "s|import '\\./_base\\.libsonnet'|import 'deps/_base.libsonnet'|g" test-output/bench.libsonnet
sed -i.bak "s|import '\\.\\./utils/templates\\.libsonnet'|import 'deps/utils/templates.libsonnet'|g" test-output/bench.libsonnet
sed -i.bak "s|import '\\.\\./\\.\\./infra-utils/version_comparisons\\.libsonnet'|import 'deps/infra-utils/version_comparisons.libsonnet'|g" test-output/bench.libsonnet
sed -i.bak "s|import 'argo-workflows-libsonnet/main\\.libsonnet'|import 'deps/vendor/argo-workflows-libsonnet/main.libsonnet'|g" test-output/bench.libsonnet
sed -i.bak "s|import 'github\\.com/jsonnet-libs/xtd/url\\.libsonnet'|import 'deps/vendor/github.com/jsonnet-libs/xtd/url.libsonnet'|g" test-output/bench.libsonnet
rm test-output/bench.libsonnet.bak

echo "🔍 Rewritten imports in generated libsonnet:"
head -10 test-output/bench.libsonnet

# Create test workflow using generated libsonnet
echo "📝 Creating test workflow using generated libsonnet..."
cat > test-workflow.jsonnet << 'EOF'
local bench = import 'test-output/bench.libsonnet';

// Create an Argo workflow that uses the bench libsonnet
{
  apiVersion: 'argoproj.io/v1alpha1',
  kind: 'Workflow',
  metadata: {
    name: 'bench-smoke-test',
    namespace: 'default',
  },
  spec: {
    entrypoint: 'bench-step',
    templates: [
      bench('bench-step').withBenchTest('http://localhost:3000', {
        testRunner: 'k6',
        path: 'CI/k6',
        type: 'smoke',
      })
    ]
  }
}
EOF

# Generate Argo workflow YAML using jsonnet
echo "⚙️  Generating Argo workflow YAML..."
jsonnet -J . test-workflow.jsonnet > generated-workflow.yaml

# Display the generated workflow
echo "📄 Generated workflow:"
cat generated-workflow.yaml

# Validate the generated YAML with Argo CLI
echo "✅ Validating workflow with Argo CLI..."
argo lint --offline --strict --kinds=workflows generated-workflow.yaml

echo ""
echo "🎉 Local Argo workflow validation completed successfully!"
echo "💡 Generated files:"
echo "   - test-output/bench.libsonnet"
echo "   - test-workflow.jsonnet" 
echo "   - generated-workflow.yaml"