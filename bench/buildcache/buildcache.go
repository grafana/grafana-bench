package buildcache

import (
	"context"
	"fmt"
	"io"
	"os"
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
	bucketName  string
	remoteCache bool
}

func NewBuildCache(ctx context.Context, localDir, credPath, bucketName string) (*BuildCache, error) {
	fmt.Println("build-cache: using local directory:", localDir)

	// If we don't have a bucket name, don't use remote object store for build cache
	if bucketName == "" {
		fmt.Println("build-cache: no remote object store defined:", localDir)
		return &BuildCache{
			LocalDir: localDir,
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
		remoteCache: bucketName != "",
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
	if !bc.remoteCache {
		return false, nil
	}

	return bc.DownloadRemoteBuild(ctx, artifactName)
}

func (bc *BuildCache) CacheBuild(ctx context.Context, buildPath, artifactName string) error {
	fmt.Println("build-cache: caching build")
	if err := sh.RunV("cp", buildPath, bc.DiskPath(artifactName)); err != nil {
		return err
	}

	if bc.remoteCache {
		fmt.Println("build-cache: uploading build to remote cache")
		return bc.UploadRemoteBuild(ctx, buildPath, artifactName)
	}

	return nil
}

// Returns path to artifact on disk
func (bc *BuildCache) DiskPath(artifactName string) string {
	return path.Join(bc.LocalDir, artifactName)
}

// Returns object name in bucket
func (bc *BuildCache) RemotePath(artifactName string) string {
	return fmt.Sprintf("builds/%s", artifactName)
}

// Returns object for given artifactName
func (bc *BuildCache) GetObjectHandleFromGrafanaBuildName(artifactName string) *storage.ObjectHandle {
	return bc.Bucket.Object(bc.RemotePath(artifactName))
}

// Sends a build to remote cache
func (bc *BuildCache) UploadRemoteBuild(ctx context.Context, buildPath, artifactName string) error {
	obj := bc.GetObjectHandleFromGrafanaBuildName(artifactName)
	exists, err := ObjectExists(ctx, obj)
	if err != nil {
		return fmt.Errorf("build-cache: Error contacting remote cache: %s", err)
	}

	if exists {
		return nil
	}

	if err := WriteToBucket(ctx, buildPath, obj); err != nil {
		return fmt.Errorf("build-cache: error writing build to bucket: %w", err)
	}

	return nil
}

// Downloads a build from remote cache
func (bc *BuildCache) DownloadRemoteBuild(ctx context.Context, artifactName string) (bool, error) {
	// attempt to get it from the build cache
	obj := bc.GetObjectHandleFromGrafanaBuildName(artifactName)

	// check to see if it exists
	exists, err := ObjectExists(ctx, obj)
	if err != nil {
		return false, fmt.Errorf("Error contacting build cache: %s", err)
	}
	if !exists {
		fmt.Println("build-cache: object not found:", obj.ObjectName())
		return false, nil
	}

	// if exists, download
	fmt.Println("build-cache: object found, downloading")
	if err := WriteToLocal(ctx, bc.DiskPath(artifactName), obj); err != nil {
		return false, fmt.Errorf("Error retrieving build artifact from bucket: %w", err)
	}

	return true, nil
}

// Wrapper for determining whether object exists
func ObjectExists(ctx context.Context, obj *storage.ObjectHandle) (bool, error) {
	_, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

// Write file from local to bucket
func WriteToBucket(ctx context.Context, filepath string, obj *storage.ObjectHandle) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	w := obj.NewWriter(ctx)

	if _, err := io.Copy(w, file); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return nil
}

// Writes file from bucket to local
func WriteToLocal(ctx context.Context, filepath string, obj *storage.ObjectHandle) error {
	r, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return err
	}

	return nil
}

// Reads file from bucket into memory as string
func ReadFromBucket(ctx context.Context, obj *storage.ObjectHandle) (string, error) {
	r, err := obj.NewReader(ctx)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Read the data from the reader into a byte slice
	data, err := io.ReadAll(r)
	if err != nil {
		return "", nil
	}

	return string(data), nil
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
