package buildcache

import (
	"path"
	"strings"

	"cloud.google.com/go/storage"
)

type CacheObjectType string

var (
	TypeBuild CacheObjectType = "build"
	TypeINI   CacheObjectType = "INI"
)

func (ct CacheObjectType) String() string {
	switch ct {
	case TypeBuild:
		return "build"
	case TypeINI:
		return "ini"
	default:
		return "Unknown"
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

// Returns object handle for given artifactName
func (bc *BuildCache) GetObjectHandle(ct CacheObjectType, artifactName string) *storage.ObjectHandle {
	return bc.Bucket.Object(bc.RemotePath(ct, artifactName))
}
