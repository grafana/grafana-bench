package buildcache

import (
	"fmt"
	"path"
	"strings"

	"cloud.google.com/go/storage"
)

type CacheObjectType string

var (
	// A grafana executable
	TypeBuild CacheObjectType = "build"
	// A default.ini file required to boot a build of Grafana
	TypeINI CacheObjectType = "INI"
	// A bundle used to provision grafana to a VM
	TypeBundle CacheObjectType = "bundle"
	// A test suite executed on remote k6 vm
	TypeTestBundle CacheObjectType = "testBundle"
)

func (ct CacheObjectType) String() string {
	switch ct {
	case TypeBuild:
		return "build"
	case TypeINI:
		return "ini"
	case TypeBundle:
		return "bundle"
	case TypeTestBundle:
		return "testBundle"
	default:
		panic("Unknown CacheObjectType")
	}
}

// Returns prefix used to organize object types on disk or in bucket
func (ct CacheObjectType) StorePrefix() string {
	// build -> builds
	return strings.ToLower(ct.String()) + "s"
}

// Returns object name in bucket
func (bc *BuildCache) RemotePath(ct CacheObjectType, artifactName string) string {
	return path.Join(ct.StorePrefix(), artifactName)
}

// Returns path to artifact on disk
func (bc *BuildCache) DiskPath(ct CacheObjectType, artifactName string) string {
	return path.Join(bc.LocalDir, ct.StorePrefix(), artifactName)
}

// Returns directory on disk where cacheObjectType is stored
func (bc *BuildCache) DiskDirectory(ct CacheObjectType) string {
	return path.Join(bc.LocalDir, ct.StorePrefix())
}

// Returns object handle for given artifactName
func (bc *BuildCache) GetObjectHandle(ct CacheObjectType, artifactName string) *storage.ObjectHandle {
	return bc.Bucket.Object(bc.RemotePath(ct, artifactName))
}

// generates the name of the build artifact for caching
// e.g. 6e4fe51fe8f0da7719eb933ef77c6e8b46dae126-darwin-arm64
func GetArtifactBuildName(grafanaGitRef, arch string) string {
	// darwin/arm64 -> darwin-arm64
	arch = strings.Replace(arch, "/", "-", -1)
	return fmt.Sprintf("%s-%s-grafana-server", grafanaGitRef, arch)
}

// generates the name of the ini artifact for caching
// e.g. 6e4fe51fe8f0da7719eb933ef77c6e8b46dae126_defaults.ini
func GetArtifactININame(grafanaGitRef string) string {
	return fmt.Sprintf("%s_defaults.ini", grafanaGitRef)
}
