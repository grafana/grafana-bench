// Base libsonnet stub for testing - provides foundation for generated libsonnet
function(name, template='') {
  local this = self,
  name:: name,
  template:: template,
  
  // Base parameters structure that matches what the generated libsonnet expects (visible field)
  parameters: {
    script: '',
    image: '',
    container_patch: '{}',
  },
  
  // Base environment variables array (visible field)
  envVars: [],
  
  // Helper methods that the generated libsonnet may call
  withEnvVars(envVars):: self {
    envVars+: envVars,
  },
  
  withContainerImage(image):: self {
    image: image,
  },
}