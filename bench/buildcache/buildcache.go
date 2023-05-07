package buildcache

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type BuildCache struct {
	Client *storage.Client
	Bucket *storage.BucketHandle
}

func NewBuildCache(ctx context.Context, credPath, bucketName string) (*BuildCache, error) {
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		return nil, err
	}

	return &BuildCache{
		Client: client,
		Bucket: client.Bucket(bucketName),
	}, nil
}

// Downloads a grafana build from object store cache
func (bc *BuildCache) DownloadGrafanaBuild(ctx context.Context, artifactName, diskDestination string) (bool, error) {
	// attempt to get it from the build cache
	obj := bc.GetObjectHandleFromGrafanaBuildName(artifactName)
	fmt.Println("build-cache: checking build cache for artifact:", obj.ObjectName())

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
	if err := WriteToLocal(ctx, diskDestination, obj); err != nil {
		return false, fmt.Errorf("Error retrieving build artifact from bucket: %w", err)
	}

	return true, nil
}

// Returns object path for given artifact
func (bc *BuildCache) GetObjectHandleFromGrafanaBuildName(artifactName string) *storage.ObjectHandle {
	name := fmt.Sprintf("builds/%s", artifactName)
	return bc.Bucket.Object(name)
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
