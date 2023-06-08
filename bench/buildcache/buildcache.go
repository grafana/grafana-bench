package buildcache

import (
	"context"
	"fmt"
	"path"

	"cloud.google.com/go/storage"
	"github.com/grafana/grafana-bench/bench/utils"
	"github.com/magefile/mage/sh"
	"google.golang.org/api/option"
)

type BuildCache struct {
	Client      *storage.Client
	Bucket      *storage.BucketHandle
	LocalDir    string
	RemoteCache bool
}

type Build struct {
	Location string
	Name     string
}

func NewBuildCache(ctx context.Context, localDir, credPath, bucketName string) (*BuildCache, error) {
	fmt.Println("build-cache: using local directory:", localDir)

	// If we don't have a bucket name, don't use remote object store for build cache
	if bucketName == "" {
		fmt.Println("build-cache: no remote object store defined:", localDir)
		return &BuildCache{
			LocalDir:    localDir,
			RemoteCache: false,
		}, nil
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, err
	}

	fmt.Println("build-cache: object store bucket:", bucketName)
	return &BuildCache{
		Client:      client,
		Bucket:      client.Bucket(bucketName),
		LocalDir:    localDir,
		RemoteCache: bucketName != "",
	}, nil
}

// Resolves a build artifact. Checks local disk and if not found checks remote bucket if defined.
// If the build is in the remote bucket, it is downloaded to the local artifacts directory and return true.
// If the build is not found on disk or in the bucket, we return false
func (bc *BuildCache) Resolve(ctx context.Context, artifactName string) (bool, error) {
	// check local
	exists, err := utils.PathExists(bc.DiskPath(artifactName))
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

	return bc.DownloadRemoteBuild(ctx, artifactName)
}

// Copies build from path specified to local cache and remote cache if
// configured
func (bc *BuildCache) Store(ctx context.Context, buildPath, artifactName string) error {
	fmt.Println("build-cache: caching build")
	if err := sh.RunV("cp", buildPath, bc.DiskPath(artifactName)); err != nil {
		return err
	}

	if bc.RemoteCache {
		fmt.Println("build-cache: uploading build to remote cache")
		return bc.UploadRemoteBuild(ctx, buildPath, artifactName)
	}

	return nil
}

// Gets a list of builds from the build cache
func (bc *BuildCache) List(ctx context.Context) ([]Build, error) {
	var builds []Build

	// Get local builds
	localBuilds, err := utils.GlobByPrefix(bc.LocalDir, "grafana-server-")
	if err != nil {
		return builds, err
	}
	for _, f := range localBuilds {
		builds = append(builds, Build{Location: "local", Name: f})
	}

	// Get remote builds if cache enabled
	if bc.RemoteCache {
		remoteBuilds, err := bc.ListRemoteBuilds(ctx)
		if err != nil {
			return builds, err
		}
		for _, b := range remoteBuilds {
			builds = append(builds, Build{Location: "remote", Name: b})
		}
	}

	return builds, nil
}

// Returns path to artifact on disk
func (bc *BuildCache) DiskPath(artifactName string) string {
	return path.Join(bc.LocalDir, artifactName)
}

//func main() {
//  ctx := context.Background()
//  credPath := "/Users/jeff/projects/g/bench/GCP-infra-manager-828bbfa6f427.json"

//  bc, err := NewBuildCache(ctx, credPath, "bench-builds")
//  if err != nil {
//    panic(err)
//  }

//  // check if object exists as a test
//  obj := bc.Bucket.Object("test.txt")
//  exists, err := ObjectExists(ctx, obj)
//  if err != nil {
//    fmt.Println("There was an error:", err)
//  }

//  if exists {
//    fmt.Println("Exists!")
//    // read file
//  } else {
//    fmt.Println("NotExists!")
//    //write file
//    if err := WriteToBucket(ctx, "/tmp/test.txt", obj); err != nil {
//      fmt.Println("error writing file:", err)
//      return
//    } else {
//      fmt.Println("file added to object store")
//    }
//  }

//  str, err := ReadFromBucket(ctx, obj)
//  if err != nil {
//    fmt.Println("error reading file:", err)
//    return
//  }

//  fmt.Println("read file:", str)
//}
