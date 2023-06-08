package buildcache

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// Returns a pattern used for querying GCS that matches the RemotePath
// function located next to buildcache.RemotePath so we can keep them in sync
func (bc *BuildCache) BuildPattern() string {
	return "builds/"
}

// Returns object name in bucket
func (bc *BuildCache) RemotePath(artifactName string) string {
	return fmt.Sprintf("builds/%s", artifactName)
}

// Returns object for given artifactName
func (bc *BuildCache) GetObjectHandleFromGrafanaBuildName(artifactName string) *storage.ObjectHandle {
	return bc.Bucket.Object(bc.RemotePath(artifactName))
}

// Gets list of builds from remote cache
func (bc *BuildCache) ListRemoteBuilds(ctx context.Context) ([]string, error) {
	var builds []string
	query := &storage.Query{Prefix: bc.BuildPattern()}
	//query.SetAttrSelection([]string{"Name"})
	it := bc.Bucket.Objects(ctx, query)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("Failed to iterate objects: %v\n", err)
		}
		name := strings.TrimPrefix(objAttrs.Name, bc.BuildPattern())
		builds = append(builds, name)
	}
	return builds, nil
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
