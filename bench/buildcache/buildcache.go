package buildcache

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"cloud.google.com/go/storage"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
	"google.golang.org/api/option"
)

type BuildCache struct {
	LocalDir    string
	Client      *storage.Client
	Bucket      *storage.BucketHandle
	BucketName  string
	RemoteCache bool
}

type BuildRef struct {
	Location string
	Name     string
}

func NewBuildCache(ctx context.Context, localDir, credPath, bucketName string) (*BuildCache, error) {
	log.Println("build-cache: using local directory:", localDir)
	// ensure build cache directory exists
	err := os.MkdirAll(localDir, 0755)
	if err != nil {
		return nil, err
	}

	// If we don't have a bucket name, don't use remote object store for build cache
	if bucketName == "" {
		log.Println("build-cache: no remote store defined")
		return &BuildCache{
			LocalDir:    localDir,
			RemoteCache: false,
		}, nil
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, err
	}

	log.Println("build-cache: using remote store bucket:", bucketName)
	return &BuildCache{
		Client:      client,
		Bucket:      client.Bucket(bucketName),
		BucketName:  bucketName,
		LocalDir:    localDir,
		RemoteCache: true,
	}, nil
}

// Gets an artifact and writes it to destination. Returns an error if artifact is
// not in cache
func (bc *BuildCache) Retrieve(ctx context.Context, ct CacheObjectType, artifactName, destination string) error {
	// resolve the file in the cache
	resolved, err := bc.Resolve(ctx, ct, artifactName)
	if err != nil {
		return err
	}

	if !resolved {
		return fmt.Errorf("build-cache: %s, not found in cache", artifactName)
	}

	// copy file to destination
	if err := sh.RunV("cp", bc.DiskPath(ct, artifactName), destination); err != nil {
		return err
	}

	return nil
}

// Exists checks if the object is in the remote cache
func (bc *BuildCache) RemoteExists(ctx context.Context, ct CacheObjectType, artifactName string) (bool, error) {
	obj := bc.GetObjectHandle(ct, artifactName)
	return ObjectExists(ctx, obj)
}

// Resolves a build artifact. Checks local disk and if not found checks remote bucket if defined.
// If the build is in the remote bucket, it is downloaded to the local artifacts directory and return true.
// If the build is not found on disk or in the bucket, we return false
func (bc *BuildCache) Resolve(ctx context.Context, ct CacheObjectType, artifactName string) (bool, error) {
	// check local
	exists, err := utils.PathExists(bc.DiskPath(ct, artifactName))
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	// return if no remote cache
	if !bc.RemoteCache {
		return false, nil
	}

	return bc.DownloadRemote(ctx, ct, artifactName)
}

// Writes file into local cache and remote cache
func (bc *BuildCache) StoreFile(ctx context.Context, ct CacheObjectType, srcPath, artifactName string) error {
	log.Println("build-cache: caching artifact", ct.String(), artifactName)

	// Copy to local cache if not already there
	diskPath := bc.DiskPath(ct, artifactName)

	// Ensure destination folder exists
	dir := path.Dir(diskPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// write to local cache
	if err := utils.Cp(srcPath, diskPath); err != nil {
		return err
	}

	// Copy to remote cache if configured
	if bc.RemoteCache {
		log.Println("build-cache: uploading to remote cache")
		return bc.UploadRemote(ctx, ct, srcPath, artifactName)
	}

	return nil
}

// Writes byte array to file in local cache and remote cache
func (bc *BuildCache) StoreBytes(ctx context.Context, ct CacheObjectType, body []byte, artifactName string) error {
	log.Println("build-cache: caching artifact", ct.String(), artifactName)

	// Copy to local cache if not already there
	diskPath := bc.DiskPath(ct, artifactName)

	// Ensure destination folder exists
	dir := path.Dir(diskPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	// write to local cache
	err = os.WriteFile(diskPath, body, 0755)
	if err != nil {
		return err
	}

	// Copy to remote cache if configured
	if bc.RemoteCache {
		log.Println("build-cache: uploading to remote cache")
		return bc.UploadRemote(ctx, ct, diskPath, artifactName)
	}

	return nil
}

// Gets presigned url for the object
func (bc *BuildCache) GetPresignedUrl(ctx context.Context, ct CacheObjectType, artifactName string) (string, error) {
	objectUrl := bc.RemotePath(ct, artifactName)

	log.Println("build-cache: generating presigned url for:", bc.BucketName, objectUrl)

	url, err := bc.Bucket.SignedURL(objectUrl, &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(30 * time.Minute),
	})

	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", bc.BucketName, err)
	}

	return url, nil
}

// Gets a list of builds from the build cache
func (bc *BuildCache) List(ctx context.Context, ct CacheObjectType) ([]BuildRef, error) {
	var builds []BuildRef

	// Get local builds
	localBuilds, err := utils.List(bc.DiskDirectory(ct))
	if err != nil {
		return builds, err
	}
	for _, f := range localBuilds {
		builds = append(builds, BuildRef{Location: "local", Name: f})
	}

	// Get remote builds if cache enabled
	if bc.RemoteCache {
		remoteBuilds, err := bc.ListRemote(ctx, ct)
		if err != nil {
			return builds, err
		}
		for _, b := range remoteBuilds {
			builds = append(builds, BuildRef{Location: "remote", Name: b})
		}
	}

	return builds, nil
}
