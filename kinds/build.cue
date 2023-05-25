package kinds

build: {
	name: "Build"
	group: "grafana-bench-app"
	crd: {
		scope: "Namespaced"
	}
	codegen: {
		frontend: true
		backend: true
	}
	pluralName: "Builds"
	lineage: {
		schemas: [{
			version: [0,0]
			schema: {
				spec: {
					applicationName: string|*"grafana"
					gitRepository: string
					gitRevision: string
					buildSuite?: string|*"latest"
					architecture: string
					operatingSystem: string
				}
				status: {
						artifactLocation: string
				}
			}
		}]
	}
}
