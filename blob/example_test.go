// Copyright 2018 The Go Cloud Development Kit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blob_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

func ExampleBucket_NewReader() {
	// Connect to a bucket when your program starts up.
	// This example uses the file-based implementation.
	dir, cleanup := newTempDir()
	defer cleanup()
	// Write a file to read using the bucket.
	err := ioutil.WriteFile(filepath.Join(dir, "foo.txt"), []byte("Hello, World!\n"), 0666)
	if err != nil {
		log.Fatal(err)
	}
	// Create the file-based bucket.
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Open a reader using the blob's key.
	ctx := context.Background()
	r, err := bucket.NewReader(ctx, "foo.txt", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	// The blob reader implements io.Reader, so we can use any function that
	// accepts an io.Reader.
	if _, err := io.Copy(os.Stdout, r); err != nil {
		log.Fatal(err)
	}

	// Output:
	// Hello, World!
}

func ExampleBucket_NewRangeReader() {
	// Connect to a bucket when your program starts up.
	// This example uses the file-based implementation.
	dir, cleanup := newTempDir()
	defer cleanup()
	// Write a file to read using the bucket.
	err := ioutil.WriteFile(filepath.Join(dir, "foo.txt"), []byte("Hello, World!\n"), 0666)
	if err != nil {
		log.Fatal(err)
	}
	// Create the file-based bucket.
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Open a reader using the blob's key at a specific offset at length.
	ctx := context.Background()
	r, err := bucket.NewRangeReader(ctx, "foo.txt", 1, 4, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	// The blob reader implements io.Reader, so we can use any function that
	// accepts an io.Reader.
	if _, err := io.Copy(os.Stdout, r); err != nil {
		log.Fatal(err)
	}

	// Output:
	// ello
}

func ExampleBucket_NewWriter() {
	// Connect to a bucket when your program starts up.
	// This example uses the file-based implementation.
	dir, cleanup := newTempDir()
	defer cleanup()
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Open a writer using the key "foo.txt" and the default options.
	ctx := context.Background()
	w, err := bucket.NewWriter(ctx, "foo.txt", nil)
	if err != nil {
		log.Fatal(err)
	}

	// The blob writer implements io.Writer, so we can use any function that
	// accepts an io.Writer. A writer must always be closed.
	_, printErr := fmt.Fprintln(w, "Hello, World!")
	closeErr := w.Close()
	if printErr != nil {
		log.Fatal(printErr)
	}
	if closeErr != nil {
		log.Fatal(closeErr)
	}
	// Copy the written blob to stdout.
	r, err := bucket.NewReader(ctx, "foo.txt", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	if _, err := io.Copy(os.Stdout, r); err != nil {
		log.Fatal(err)
	}
	// Since we didn't specify a WriterOptions.ContentType for NewWriter, blob
	// auto-determined one using http.DetectContentType.
	fmt.Println(r.ContentType())

	// Output:
	// Hello, World!
	// text/plain; charset=utf-8
}

func Example() {
	// Connect to a bucket when your program starts up.
	// This example uses the file-based implementation in fileblob, and creates
	// a temporary directory to use as the root directory.
	dir, cleanup := newTempDir()
	defer cleanup()
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatal(err)
	}

	// We now have a *blob.Bucket! We can write our application using the
	// *blob.Bucket type, and have the freedom to change the initialization code
	// above to choose a different provider later.

	// In this example, we'll write a blob and then read it.
	ctx := context.Background()
	if err := bucket.WriteAll(ctx, "foo.txt", []byte("Go Cloud Development Kit"), nil); err != nil {
		log.Fatal(err)
	}
	b, err := bucket.ReadAll(ctx, "foo.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	// Output:
	// Go Cloud Development Kit
}

func ExampleBucket_ErrorAs() {
	// This example is specific to the s3blob implementation; it demonstrates
	// access to the underlying awserr.Error type.
	// The types exposed for ErrorAs by s3blob are documented in
	// https://godoc.org/gocloud.dev/blob/s3blob#hdr-As

	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "s3://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	_, err = b.ReadAll(ctx, "nosuchfile")
	if err != nil {
		var awsErr awserr.Error
		if b.ErrorAs(err, &awsErr) {
			fmt.Println(awsErr.Code())
		}
	}
}

func ExampleBucket_List() {
	// Connect to a bucket when your program starts up.
	// This example uses the file-based implementation.
	dir, cleanup := newTempDir()
	defer cleanup()

	// Create the file-based bucket.
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create some blob objects for listing: "foo[0..4].txt".
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := bucket.WriteAll(ctx, fmt.Sprintf("foo%d.txt", i), []byte("Go Cloud Development Kit"), nil); err != nil {
			log.Fatal(err)
		}
	}

	// Iterate over them.
	// This will list the blobs created above because fileblob is strongly
	// consistent, but is not guaranteed to work on all providers.
	iter := bucket.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(obj.Key)
	}

	// Output:
	// foo0.txt
	// foo1.txt
	// foo2.txt
	// foo3.txt
	// foo4.txt
}

func ExampleBucket_List_withDelimiter() {
	// Connect to a bucket when your program starts up.
	// This example uses the file-based implementation.
	dir, cleanup := newTempDir()
	defer cleanup()

	// Create the file-based bucket.
	bucket, err := fileblob.OpenBucket(dir, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Create some blob objects in a hierarchy.
	ctx := context.Background()
	for _, key := range []string{
		"dir1/subdir/a.txt",
		"dir1/subdir/b.txt",
		"dir2/c.txt",
		"d.txt",
	} {
		if err := bucket.WriteAll(ctx, key, []byte("Go Cloud Development Kit"), nil); err != nil {
			log.Fatal(err)
		}
	}

	// list lists files in b starting with prefix. It uses the delimiter "/",
	// and recurses into "directories", adding 2 spaces to indent each time.
	// It will list the blobs created above because fileblob is strongly
	// consistent, but is not guaranteed to work on all providers.
	var list func(context.Context, *blob.Bucket, string, string)
	list = func(ctx context.Context, b *blob.Bucket, prefix, indent string) {
		iter := b.List(&blob.ListOptions{
			Delimiter: "/",
			Prefix:    prefix,
		})
		for {
			obj, err := iter.Next(ctx)
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("%s%s\n", indent, obj.Key)
			if obj.IsDir {
				list(ctx, b, obj.Key, indent+"  ")
			}
		}
	}
	list(ctx, bucket, "", "")

	// Output:
	// d.txt
	// dir1/
	//   dir1/subdir/
	//     dir1/subdir/a.txt
	//     dir1/subdir/b.txt
	// dir2/
	//   dir2/c.txt
}

func ExampleBucket_As() {
	// This example is specific to the gcsblob implementation; it demonstrates
	// access to the underlying cloud.google.com/go/storage.Client type.
	// The types exposed for As by gcsblob are documented in
	// https://godoc.org/gocloud.dev/blob/gcsblob#hdr-As

	// This URL will open the bucket "my-bucket" using default credentials.
	ctx := context.Background()
	b, err := blob.OpenBucket(ctx, "gs://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	// Access storage.Client fields via gcsClient here.
	var gcsClient *storage.Client
	if b.As(&gcsClient) {
		email, err := gcsClient.ServiceAccount(ctx, "project-name")
		if err != nil {
			log.Fatal(err)
		}
		_ = email
	} else {
		log.Println("Unable to access storage.Client through Bucket.As")
	}
}

func ExampleWriterOptions() {
	// This example is specific to the gcsblob implementation; it demonstrates
	// access to the underlying cloud.google.com/go/storage.Writer type.
	// The types exposed for As by gcsblob are documented in
	// https://godoc.org/gocloud.dev/blob/gcsblob#hdr-As

	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "gs://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	beforeWrite := func(as func(interface{}) bool) error {
		var sw *storage.Writer
		if as(&sw) {
			fmt.Println(sw.ChunkSize)
		}
		return nil
	}

	options := blob.WriterOptions{BeforeWrite: beforeWrite}
	if err := b.WriteAll(ctx, "newfile.txt", []byte("hello\n"), &options); err != nil {
		log.Fatal(err)
	}
}

func ExampleListObject_As() {
	// This example is specific to the gcsblob implementation; it demonstrates
	// access to the underlying cloud.google.com/go/storage.ObjectAttrs type.
	// The types exposed for As by gcsblob are documented in
	// https://godoc.org/gocloud.dev/blob/gcsblob#hdr-As

	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "gs://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	iter := b.List(nil)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// Access storage.ObjectAttrs via oa here.
		var oa storage.ObjectAttrs
		if obj.As(&oa) {
			_ = oa.Owner
		}
	}
}

func ExampleListOptions() {
	// This example is specific to the gcsblob implementation; it demonstrates
	// access to the underlying cloud.google.com/go/storage.Query type.
	// The types exposed for As by gcsblob are documented in
	// https://godoc.org/gocloud.dev/blob/gcsblob#hdr-As

	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "gs://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	beforeList := func(as func(interface{}) bool) error {
		// Access storage.Query via q here.
		var q *storage.Query
		if as(&q) {
			_ = q.Delimiter
		}
		return nil
	}

	iter := b.List(&blob.ListOptions{Prefix: "", Delimiter: "/", BeforeList: beforeList})
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		_ = obj
	}
}

func ExampleReader_As() {
	// This example is specific to the gcsblob implementation; it demonstrates
	// access to the underlying cloud.google.com/go/storage.Reader type.
	// The types exposed for As by gcsblob are documented in
	// https://godoc.org/gocloud.dev/blob/gcsblob#hdr-As

	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "gs://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	r, err := b.NewReader(ctx, "gopher.png", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Access storage.Reader via sr here.
	var sr storage.Reader
	if r.As(&sr) {
		_ = sr.Attrs
	}
}

func ExampleAttributes_As() {
	// This example is specific to the gcsblob implementation; it demonstrates
	// access to the underlying cloud.google.com/go/storage.ObjectAttrs type.
	// The types exposed for As by gcsblob are documented in
	// https://godoc.org/gocloud.dev/blob/gcsblob#hdr-As
	ctx := context.Background()

	b, err := blob.OpenBucket(ctx, "gs://my-bucket")
	if err != nil {
		log.Fatal(err)
	}

	attrs, err := b.Attributes(ctx, "gopher.png")
	if err != nil {
		log.Fatal(err)
	}

	var oa storage.ObjectAttrs
	if attrs.As(&oa) {
		fmt.Println(oa.Owner)
	}
}

func newTempDir() (string, func()) {
	dir, err := ioutil.TempDir("", "go-cloud-blob-example")
	if err != nil {
		panic(err)
	}
	return dir, func() { os.RemoveAll(dir) }
}
