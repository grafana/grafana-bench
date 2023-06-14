package buildcache

import (
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
)

type CacheObjectType string

var (
	BuildObj CacheObjectType = "build"
	IniObj   CacheObjectType = "INI"
)

func (ct CacheObjectType) String() string {
	switch ct {
	case BuildObj:
		return "build"
	case IniObj:
		return "ini"
	default:
		return "Unknown"
	}
}

// Returns prefix used to organize object types in bucket
func (ct CacheObjectType) ObjectStorePrefix() string {
	// build -> builds/
	return strings.ToLower(ct.String()) + "s/"
}

// Returns object name in bucket
func (bc *BuildCache) RemotePath(ct CacheObjectType, artifactName string) string {
	return fmt.Sprintf("%s/%s", ct.ObjectStorePrefix(), artifactName)
}

// Returns object handle for given artifactName
func (bc *BuildCache) GetObjectHandle(ct CacheObjectType, artifactName string) *storage.ObjectHandle {
	return bc.Bucket.Object(bc.RemotePath(ct, artifactName))
}
