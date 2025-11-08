// Base libsonnet stub for testing - provides foundation for generated libsonnet
// Returns a basic Argo workflow template structure
function(name, template='') {
  local this = self,
  
  // Template name is required by Argo
  name: name,
  
  // Use container template as default
  container: {
    image: 'alpine:latest',
    command: ['sh', '-c'],
    args: ['echo "placeholder"'],
  },
  
  // Hidden fields that generated libsonnet can access
  template:: template,
  
  // Base parameters structure that matches what the generated libsonnet expects
  parameters: {
    script: '',
    image: '',
    container_patch: '{}',
  },
  
  // Base environment variables array
  envVars: [],
  
  // Helper methods that the generated libsonnet may call
  withEnvVars(envVars):: self {
    envVars+: envVars,
  },
  
  withContainerImage(image):: self {
    image: image,
  },
}