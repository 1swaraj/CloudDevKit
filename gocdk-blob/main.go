















package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/google/subcommands"
	"genericstoragesdk/blob"


	_ "genericstoragesdk/blob/fileblob"
	_ "genericstoragesdk/blob/gcsblob"
	_ "genericstoragesdk/blob/s3blob"
)

const helpSuffix = `

  See https:
  Go CDK URLs, and sub-packages under genericstoragesdk/blob
  (https:
  for details on the blob.Bucket URL format.
`

func main() {
	os.Exit(run())
}

func run() int {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(&downloadCmd{}, "")
	subcommands.Register(&listCmd{}, "")
	subcommands.Register(&uploadCmd{}, "")
	log.SetFlags(0)
	log.SetPrefix("gocdk-blob: ")
	flag.Parse()
	return int(subcommands.Execute(context.Background()))
}

type downloadCmd struct{}

func (*downloadCmd) Name() string     { return "download" }
func (*downloadCmd) Synopsis() string { return "Output a blob to stdout" }
func (*downloadCmd) Usage() string {
	return `download <bucket URL> <key>

  Read the blob <key> from <bucket URL> and write it to stdout.

  Example:
    gocdk-blob download gs:
}

func (*downloadCmd) SetFlags(_ *flag.FlagSet) {}

func (*downloadCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 2 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	bucketURL := f.Arg(0)
	blobKey := f.Arg(1)

	
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		log.Printf("Failed to open bucket: %v\n", err)
		return subcommands.ExitFailure
	}
	defer bucket.Close()

	
	reader, err := bucket.NewReader(ctx, blobKey, nil)
	if err != nil {
		log.Printf("Failed to read %q: %v\n", blobKey, err)
		return subcommands.ExitFailure
	}
	defer reader.Close()

	
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Printf("Failed to copy data: %v\n", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

type listCmd struct {
	prefix    string
	delimiter string
}

func (*listCmd) Name() string     { return "ls" }
func (*listCmd) Synopsis() string { return "List blobs in a bucket" }
func (*listCmd) Usage() string {
	return `ls [-p <prefix>] [d <delimiter>] <bucket URL>

  List the blobs in <bucket URL>.

  Example:
    gocdk-blob ls -p "subdir/" gs:
}

func (cmd *listCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&cmd.prefix, "p", "", "prefix to match")
	f.StringVar(&cmd.delimiter, "d", "/", "directory delimiter; empty string returns flattened listing")
}

func (cmd *listCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	bucketURL := f.Arg(0)


	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		log.Printf("Failed to open bucket: %v\n", err)
		return subcommands.ExitFailure
	}
	defer bucket.Close()

	opts := blob.ListOptions{
		Prefix:    cmd.prefix,
		Delimiter: cmd.delimiter,
	}
	iter := bucket.List(&opts)
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Failed to list: %v", err)
			return subcommands.ExitFailure
		}
		fmt.Println(obj.Key)
	}
	return subcommands.ExitSuccess
}

type uploadCmd struct{}

func (*uploadCmd) Name() string     { return "upload" }
func (*uploadCmd) Synopsis() string { return "Upload a blob from stdin" }
func (*uploadCmd) Usage() string {
	return `upload <bucket URL> <key>

  Read from stdin and write to the blob <key> in <bucket URL>.

  Example:
    cat foo.txt | gocdk-blob upload gs:
}

func (*uploadCmd) SetFlags(_ *flag.FlagSet) {}

func (*uploadCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) (status subcommands.ExitStatus) {
	if f.NArg() != 2 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	bucketURL := f.Arg(0)
	blobKey := f.Arg(1)

	
	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		log.Printf("Failed to open bucket: %v\n", err)
		return subcommands.ExitFailure
	}
	defer bucket.Close()

	
	writer, err := bucket.NewWriter(ctx, blobKey, nil)
	if err != nil {
		log.Printf("Failed to write %q: %v\n", blobKey, err)
		return subcommands.ExitFailure
	}
	defer func() {
		if err := writer.Close(); err != nil && status == subcommands.ExitSuccess {
			log.Printf("closing the writer: %v", err)
			status = subcommands.ExitFailure
		}
	}()

	
	_, err = io.Copy(writer, os.Stdin)
	if err != nil {
		log.Printf("Failed to copy data: %v\n", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
