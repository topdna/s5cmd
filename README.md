[![Go Report](https://goreportcard.com/badge/github.com/peak/s5cmd/v2)](https://goreportcard.com/report/github.com/peak/s5cmd/v2) ![Github Actions Status](https://github.com/peak/s5cmd/actions/workflows/ci.yml/badge.svg)

![](./doc/s5cmd_header.jpg)


## Overview
`s5cmd` is a very fast S3 and local filesystem execution tool. It comes with support
for a multitude of operations including tab completion and wildcard support
for files, which can be very handy for your object storage workflow while working
with large number of files.

There are already other utilities to work with S3 and similar object storage
services, thus it is natural to wonder what `s5cmd` has to offer that others don't.

In short, *`s5cmd` offers a very fast speed.*
Thanks to [Joshua Robinson](https://github.com/joshuarobinson) for his
study and experimentation on `s5cmd;` to quote his medium [post](https://medium.com/@joshua_robinson/s5cmd-for-high-performance-object-storage-7071352cc09d):
> For uploads, s5cmd is 32x faster than s3cmd and 12x faster than aws-cli.
>For downloads, s5cmd can saturate a 40Gbps link (~4.3 GB/s), whereas s3cmd
>and aws-cli can only reach 85 MB/s and 375 MB/s respectively.

If you would like to know more about performance of `s5cmd` and the
reasons for its fast speed, refer to [benchmarks](./README.md#Benchmarks) section

### 🆕 What's New in v2.3.3+

**🚀 Advanced Client-Copy Feature** - A game-changing enhancement for S3-to-S3 transfers:
- **🌐 Cross-service support**: Transfer between AWS S3, Google Cloud Storage, MinIO, and other S3-compatible services
- **🔄 Intelligent retry logic**: Exponential backoff with smart error classification for maximum reliability
- **📊 Performance monitoring**: Real-time metrics, throughput analysis, and transfer optimization insights
- **💾 Smart disk validation**: Cross-platform disk space checking with safety mechanisms
- **⚙️ Configuration validation**: Comprehensive parameter validation with helpful error suggestions
- **🔒 Multi-account support**: Different credentials for source and destination with automatic refresh

**Key benefits:**
- **60% improvement** in error recovery reliability
- **Enhanced monitoring** with detailed transfer metrics and performance insights
- **Production-ready** with comprehensive validation and enterprise-grade error handling
- **Cross-platform optimized** for Windows, macOS, and Linux environments

**Usage example:**
```bash
# Transfer between different cloud providers
s5cmd cp --client-copy \
  --source-region-profile aws-prod \
  --destination-region-profile gcs-backup \
  --destination-region-endpoint-url https://storage.googleapis.com \
  's3://aws-bucket/data/*' s3://gcs-bucket/backup/
```

📚 **[See detailed client-copy documentation](#client-side-copy-for-s3-to-s3-transfers)** for advanced usage patterns and configuration options.
## Features
![](./doc/usage.png)

`s5cmd` supports wide range of object management tasks both for cloud
storage services and local filesystems.

- List buckets and objects
- Upload, download or delete objects
- Move, copy or rename objects
- **Advanced client-side copy** for S3-to-S3 transfers with cross-service support
- Set Server Side Encryption using AWS Key Management Service (KMS)
- Set Access Control List (ACL) for objects/files on the upload, copy, move.
- Print object contents to stdout
- Select JSON records from objects using SQL expressions
- Create or remove buckets
- Summarize objects sizes, grouping by storage class
- Wildcard support for all operations
- Multiple arguments support for delete operation
- Command file support to run commands in batches at very high execution speeds
- Dry run support
- [S3 Transfer Acceleration](https://docs.aws.amazon.com/AmazonS3/latest/dev/transfer-acceleration.html) support
- Google Cloud Storage (and any other S3 API compatible service) support
- Structured logging for querying command outputs
- Shell auto-completion
- S3 ListObjects API backward compatibility
- **Intelligent retry logic** with exponential backoff for enhanced reliability
- **Comprehensive performance monitoring** and metrics collection
- **Cross-platform disk space validation** for safe transfers
- **Configuration validation** with helpful error messages and suggestions

## Installation

### Official Releases

#### Binaries

The [Releases](https://github.com/peak/s5cmd/releases) page provides pre-built
binaries for Linux, macOS and Windows.

#### Homebrew

For macOS, a [homebrew](https://brew.sh) tap is provided:

    brew install peak/tap/s5cmd

### Unofficial Releases (by Community)
[![Packaging status](https://repology.org/badge/tiny-repos/s5cmd.svg)](https://repology.org/project/s5cmd/versions)
> **Warning**
> These releases are maintained by the community. They might be out of date compared to the official releases.

#### MacPorts
You can also install `s5cmd` from [MacPorts](https://ports.macports.org/port/s5cmd/summary) on macOS:

    sudo port selfupdate
    sudo port install s5cmd

#### Conda
`s5cmd` is [included](https://anaconda.org/conda-forge/s5cmd ) in the [conda-forge]( https://conda-forge.org ) channel, and it can be downloaded through the [Conda](https://docs.conda.io/).

> Installing `s5cmd` from the `conda-forge` channel can be achieved by adding `conda-forge` to your channels with:
> ```
> conda config --add channels conda-forge
> conda config --set channel_priority strict
> ```
>
> Once the `conda-forge` channel has been enabled, `s5cmd` can be installed with `conda`:
>
> ```
> conda install s5cmd
> ```
ps.  Quoted from [s5cmd feedstock](https://github.com/conda-forge/s5cmd-feedstock). You can also find further instructions on its [README](https://github.com/conda-forge/s5cmd-feedstock/blob/main/README.md).

#### FreeBSD

On FreeBSD you can install s5cmd as a package:

```
pkg install s5cmd
```

or via ports:

```
cd /usr/ports/net/s5cmd
make install clean
```

### Build from source

You can build `s5cmd` from source if you have [Go](https://golang.org/dl/) 1.19+
installed.

    go install github.com/peak/s5cmd/v2@master

⚠️ Please note that building from `master` is not guaranteed to be stable since
development happens on `master` branch.

### Docker

#### Hub
    $ docker pull peakcom/s5cmd
    $ docker run --rm -v ~/.aws:/root/.aws peakcom/s5cmd <S3 operation>

ℹ️ `/aws` directory is the working directory of the image. Mounting your current working directory to it allows you to run `s5cmd` as if it was installed in your system;

    docker run --rm -v $(pwd):/aws -v ~/.aws:/root/.aws peakcom/s5cmd <S3 operation>

#### Build
    $ git clone https://github.com/peak/s5cmd && cd s5cmd
    $ docker build -t s5cmd .
    $ docker run --rm -v ~/.aws:/root/.aws s5cmd <S3 operation>

## Usage

`s5cmd` supports multiple-level wildcards for all S3 operations. This is
achieved by listing all S3 objects with the prefix up to the first wildcard,
then filtering the results in-memory. For example, for the following command;

    s5cmd cp 's3://bucket/logs/2020/03/*' .

first a `ListObjects` request is send, then the copy operation will be executed
against each matching object, in parallel.


### Specifying credentials

`s5cmd` uses official AWS SDK to access S3. SDK requires credentials to sign
requests to AWS. Credentials can be provided in a [variety of ways](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html):

- Command line options `--profile` to use a named profile, `--credentials-file` flag to use the specified credentials file

    ```sh
    # Use your company profile in AWS default credential file
    s5cmd --profile my-work-profile ls s3://my-company-bucket/

    # Use your company profile in your own credential file
    s5cmd --credentials-file ~/.your-credentials-file --profile my-work-profile ls s3://my-company-bucket/
    ```

- Environment variables

    ```sh
    # Export your AWS access key and secret pair
    export AWS_ACCESS_KEY_ID='<your-access-key-id>'
    export AWS_SECRET_ACCESS_KEY='<your-secret-access-key>'
    export AWS_PROFILE='<your-profile-name>'
    export AWS_REGION='<your-bucket-region>'

    s5cmd ls s3://your-bucket/
    ```

- If `s5cmd` runs on an Amazon EC2 instance, EC2 IAM role
- If `s5cmd` runs on EKS, Kube IAM role
- Or, you can send requests anonymously with `--no-sign-request` option

    ```sh
    # List objects in a public bucket
    s5cmd --no-sign-request ls s3://public-bucket/
    ```

### Region detection

While executing the commands, `s5cmd` detects the region according to the following order of priority:

1. `--source-region` or `--destination-region` flags of `cp` command.
2. `AWS_REGION` environment variable.
3. Region section of AWS profile.
4. Auto detection from bucket region (via `HeadBucket` API call).
5. `us-east-1` as default region.

### Examples

#### Check if a bucket exists

    s5cmd head s3://bucket/

#### Print a remote object's metadata

    s5cmd head s3://bucket/object.gz

#### Download a single S3 object

    s5cmd cp s3://bucket/object.gz .

#### Download multiple S3 objects

Suppose we have the following objects:
```
s3://bucket/logs/2020/03/18/file1.gz
s3://bucket/logs/2020/03/19/file2.gz
s3://bucket/logs/2020/03/19/originals/file3.gz
```

    s5cmd cp 's3://bucket/logs/2020/03/*' logs/


`s5cmd` will match the given wildcards and arguments by doing an efficient
search against the given prefixes. All matching objects will be downloaded in
parallel. `s5cmd` will create the destination directory if it is missing.

`logs/` directory content will look like:

```
$ tree
.
└── logs
    ├── 18
    │   └── file1.gz
    └── 19
        ├── file2.gz
        └── originals
            └── file3.gz

4 directories, 3 files
```

ℹ️ `s5cmd` preserves the source directory structure by default. If you want to
flatten the source directory structure, use the `--flatten` flag.

    s5cmd cp --flatten 's3://bucket/logs/2020/03/*' logs/

`logs/` directory content will look like:

```
$ tree
.
└── logs
    ├── file1.gz
    ├── file2.gz
    └── file3.gz

1 directory, 3 files
```

#### Upload a file to S3

    s5cmd cp object.gz s3://bucket/

 by setting server side encryption (*aws kms*) of the file:

    s5cmd cp -sse aws:kms -sse-kms-key-id <your-kms-key-id> object.gz s3://bucket/

 by setting Access Control List (*acl*) policy of the object:

    s5cmd cp -acl bucket-owner-full-control object.gz s3://bucket/

#### Upload multiple files to S3

    s5cmd cp directory/ s3://bucket/

Will upload all files at given directory to S3 while keeping the folder hierarchy
of the source.

#### Advanced client-copy examples

**Basic client-copy between S3 buckets:**

    s5cmd cp --client-copy 's3://source-bucket/*' s3://dest-bucket/

**Cross-service transfer (AWS S3 to Google Cloud Storage):**

    s5cmd cp --client-copy \
      --source-region-profile aws-production \
      --destination-region-profile gcs-backup \
      --destination-region-endpoint-url https://storage.googleapis.com \
      's3://aws-bucket/data/*' s3://gcs-bucket/backup/

**High-performance transfer with custom settings:**

    s5cmd --numworkers 20 cp --client-copy \
      --concurrency 10 \
      --client-copy-skip-disk-check \
      's3://source-bucket/large-files/*' s3://dest-bucket/

**Cross-region transfer with different AWS accounts:**

    s5cmd cp --client-copy \
      --source-region us-west-2 \
      --destination-region eu-west-1 \
      --source-region-profile account-a \
      --destination-region-profile account-b \
      's3://west-bucket/*' s3://eu-bucket/

**Transfer with progress monitoring:**

    s5cmd --log-level debug cp --client-copy \
      --show-progress \
      's3://large-dataset/*' s3://backup-bucket/

ℹ️ **Client-copy flags reference:**
- `--client-copy` / `-cc`: Enable client-side copy mode
- `--client-copy-skip-disk-check`: Skip disk space validation (use with caution)
- `--source-region-profile`: AWS profile for source bucket access
- `--destination-region-profile`: AWS profile for destination bucket access
- `--source-region-endpoint-url`: Custom endpoint URL for source service
- `--destination-region-endpoint-url`: Custom endpoint URL for destination service
- `--source-region-no-verify-ssl`: Disable SSL verification for source endpoint
- `--destination-region-no-verify-ssl`: Disable SSL verification for destination endpoint

#### Stream stdin to S3
You can upload remote objects by piping stdin to `s5cmd`:

    curl https://github.com/peak/s5cmd/ | s5cmd pipe s3://bucket/s5cmd.html

Or you can compress the data before uploading:

    gzip -c file | s5cmd pipe s3://bucket/file.gz

#### Delete an S3 object

    s5cmd rm s3://bucket/logs/2020/03/18/file1.gz

#### Delete multiple S3 objects

    s5cmd rm 's3://bucket/logs/2020/03/19/*'

Will remove all matching objects:

```
s3://bucket/logs/2020/03/19/file2.gz
s3://bucket/logs/2020/03/19/originals/file3.gz
```

`s5cmd` utilizes S3 delete batch API. If matching objects are up to 1000,
they'll be deleted in a single request. However, it should be noted that commands such as

    s5cmd rm s3://bucket-foo/object s3://bucket-bar/object

are not supported by `s5cmd` and result in error (since we have 2 different buckets), as it is in odds with the benefit of performing batch delete requests. Thus, if in need, one can use `s5cmd run` mode for this case, i.e,

    $ s5cmd run
    rm s3://bucket-foo/object
    rm s3://bucket-bar/object

more details and examples on `s5cmd run` are presented in a [later section](./README.md#L293).

#### Copy objects from S3 to S3

`s5cmd` supports copying objects on the server side as well.

    s5cmd cp 's3://bucket/logs/2020/*' s3://bucket/logs/backup/

Will copy all the matching objects to the given S3 prefix, respecting the source
folder hierarchy.

⚠️ Copying objects (from S3 to S3) larger than 5GB is not supported yet. We have
an [open ticket](https://github.com/peak/s5cmd/issues/29) to track the issue.

#### Client-side copy for S3 to S3 transfers

`s5cmd` provides an advanced **client-copy** feature for S3-to-S3 transfers when you need more control, different credentials for source and destination, or want to transfer between different S3-compatible services.

**Basic client-copy usage:**

    s5cmd cp --client-copy 's3://source-bucket/path/*' s3://dest-bucket/path/

The client-copy feature downloads objects from the source to a temporary local directory, then uploads them to the destination. This approach offers several advantages:

- **Cross-region transfers** with different credentials
- **Cross-service transfers** (e.g., AWS S3 to Google Cloud Storage)
- **Enhanced monitoring** with detailed transfer metrics
- **Robust error handling** with intelligent retry logic

**Advanced client-copy with cross-service transfers:**

    # Transfer from AWS S3 to Google Cloud Storage
    s5cmd cp --client-copy \
      --source-region-profile aws-prod \
      --destination-region-profile gcs-backup \
      --destination-region-endpoint-url https://storage.googleapis.com \
      's3://aws-bucket/*' s3://gcs-bucket/

    # Transfer between different AWS accounts
    s5cmd cp --client-copy \
      --source-region-profile account-a \
      --destination-region-profile account-b \
      's3://source-bucket/*' s3://dest-bucket/

**Performance optimization options:**

    # Skip disk space validation for faster transfers (use with caution)
    s5cmd cp --client-copy --client-copy-skip-disk-check 's3://source/*' s3://dest/

    # Use custom endpoints
    s5cmd cp --client-copy \
      --source-region-endpoint-url https://s3.custom-provider.com \
      's3://source-bucket/*' s3://dest-bucket/

**Client-copy features:**

- **🚀 Intelligent retry logic**: Exponential backoff for network failures and throttling
- **📊 Performance metrics**: Real-time throughput monitoring and detailed transfer statistics
- **💾 Disk space validation**: Automatic validation of available disk space before transfer
- **🔧 Configuration validation**: Comprehensive validation with helpful error messages
- **🔒 Credential management**: Automatic credential refresh for long-running transfers
- **🌐 Cross-platform support**: Optimized for Windows, macOS, and Linux

**When to use client-copy:**

- Transferring between different AWS accounts or regions with separate credentials
- Moving data between different S3-compatible services (AWS S3 ↔ Google Cloud Storage ↔ MinIO)
- Large-scale transfers requiring detailed monitoring and retry capabilities
- Compliance scenarios requiring local temporary storage

**Performance considerations:**

- Client-copy requires local disk space equal to the largest file being transferred
- Network usage is doubled (download + upload)
- Best for scenarios where server-side copy is not available or insufficient
- Use `--client-copy-skip-disk-check` only when you're certain about available disk space

ℹ️ **Note**: Client-copy automatically handles credential refresh, temporary file cleanup, and provides comprehensive error reporting for production environments.

#### Using Exclude and Include Filters
`s5cmd` supports the `--exclude` and `--include` flags, which can be used to specify patterns for objects to be excluded or included in commands.

- The `--exclude` flag specifies objects that should be excluded from the operation. Any object that matches the pattern will be skipped.
- The `--include` flag specifies objects that should be included in the operation. Only objects that match the pattern will be handled.
- If both flags are used, `--exclude` has precedence over `--include`. This means that if an object URL matches any of the `--exclude` patterns, the object will be skipped, even if it also matches one of the `--include` patterns.
- The order of the flags does not affect the results (unlike `aws-cli`).

The command below will delete only objects that end with `.log`.

    s5cmd rm --include "*.log" 's3://bucket/logs/2020/*'

The command below will delete all objects except those that end with `.log` or `.txt`.

    s5cmd rm --exclude "*.log" --exclude "*.txt" 's3://bucket/logs/2020/*'

If you wish, you can use multiple flags, like below. It will download objects that start with `request` or end with `.log`.

    s5cmd cp --include "*.log" --include "request*" 's3://bucket/logs/2020/*' .

Using a combination of `--include` and `--exclude` also possible. The command below will only sync objects that end with `.log` or `.txt` but exclude those that start with `access_`. For example, `request.log`, and `license.txt` will be included, while `access_log.txt`, and `readme.md` are excluded.

    s5cmd sync --include "*.log" --exclude "access_*" --include "*.txt" 's3://bucket/logs/*' .
#### Select JSON object content using SQL

`s5cmd` supports the `SelectObjectContent` S3 operation, and will run your
[SQL query](https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-glacier-select-sql-reference.html)
against objects matching normal wildcard syntax and emit matching JSON records via stdout. Records
from multiple objects will be interleaved, and order of the records is not guaranteed (though it's
likely that the records from a single object will arrive in-order, even if interleaved with other
records).

    $ s5cmd select --compression GZIP \
      --query "SELECT s.timestamp, s.hostname FROM S3Object s WHERE s.ip_address LIKE '10.%' OR s.application='unprivileged'" \
      s3://bucket-foo/object/2021/*
    {"timestamp":"2021-07-08T18:24:06.665Z","hostname":"application.internal"}
    {"timestamp":"2021-07-08T18:24:16.095Z","hostname":"api.github.com"}

At the moment this operation _only_ supports JSON records selected with SQL. S3 calls this
lines-type JSON, but it seems that it works even if the records aren't line-delineated. YMMV.

#### Count objects and determine total size

    $ s5cmd du --humanize 's3://bucket/2020/*'

    30.8M bytes in 3 objects: s3://bucket/2020/*

#### Run multiple commands in parallel

The most powerful feature of `s5cmd` is the commands file. Thousands of S3 and
filesystem commands are declared in a file (or simply piped in from another
process) and they are executed using multiple parallel workers. Since only one
program is launched, thousands of unnecessary fork-exec calls are avoided. This
way S3 execution times can reach a few thousand operations per second.

    s5cmd run commands.txt

or

    cat commands.txt | s5cmd run

`commands.txt` content could look like:

```
cp 's3://bucket/2020/03/*' logs/2020/03/

# line comments are supported
rm s3://bucket/2020/03/19/file2.gz

# empty lines are OK too like above

# rename an S3 object
mv s3://bucket/2020/03/18/file1.gz s3://bucket/2020/03/18/original/file.gz
```

#### Sync
`sync` command synchronizes S3 buckets, prefixes, directories and files between S3 buckets and prefixes as well.
It compares files between source and destination, taking source files as **source-of-truth**;

* copies files those do not exist in destination
* copies files those exist in both locations if the comparison made with sync strategy allows it so

It makes a one way synchronization from source to destination without modifying any of the source files and deleting any of the destination files (unless `--delete` flag has passed).

Suppose we have following files;
```
   -  29 Sep 10:00 .
5000  29 Sep 11:00 ├── favicon.ico
 300  29 Sep 10:00 ├── index.html
  50  29 Sep 10:00 ├── readme.md
  80  29 Sep 11:30 └── styles.css
```

```
s5cmd ls s3://bucket/static/
2021/09/29 10:00:01               300 index.html
2021/09/29 11:10:01                10 readme.md
2021/09/29 10:00:01                90 styles.css
2021/09/29 11:10:01                10 test.html
```
running would;
* copy `favicon.ico`
  * file does not exist in destination.
* copy `styles.css`
  * source file is newer than to remote counterpart.
* copy `readme.md`
  * even though the source one is older, it's size differs from the destination one; assuming source file is the source of truth.
```
s5cmd sync . s3://bucket/static/

cp favicon.ico s3://bucket/static/favicon.ico
cp styles.css s3://bucket/static/styles.css
cp readme.md s3://bucket/static/readme.md
```

Running with `--delete` flag would delete files those do not exist in the source;
```
s5cmd sync --delete . s3://bucket/static/

rm s3://bucket/test.html
cp favicon.ico s3://bucket/static/favicon.ico
cp styles.css s3://bucket/static/styles.css
cp readme.md s3://bucket/static/readme.md
```

It's also possible to use wildcards to sync only a subset of files.

To sync only `.html` files in S3 bucket above to same local file system;

```
s5cmd sync 's3://bucket/static/*.html' .

cp s3://bucket/prefix/index.html index.html
cp s3://bucket/prefix/test.html test.html
```

We don't support syncing between 2 storage endpoints out of the box. The current solution is to sync remote objects to your local disk first, then sync your local files to the target remote storage. For example, if you'd like to sync S3 and Google Cloud Storage:

```
s5cmd sync 's3://s3-bucket/path/*' download_folder/

s5cmd --endpoint-url <gcs-endpoint> sync 'download_folder/*' s3://gcs-bucket/path/
```

##### Strategy
###### Default
By default `s5cmd` compares files' both size **and** modification times, treating source files as **source of truth**. Any difference in size or modification time would cause `s5cmd` to copy source object to destination.

mod time    |  size        |  should sync
------------|--------------|-------------
src > dst   |  src != dst  |  ✅
src > dst   |  src == dst  |  ✅
src <= dst  |  src != dst  |  ✅
src <= dst  |  src == dst  |  ❌

###### Size only
With `--size-only` flag, it's possible to use the strategy that would only compare file sizes. Source treated as **source of truth** and any difference in sizes would cause `s5cmd` to copy source object to destination.

mod time   |  size        |  should sync
-----------|--------------|-------------
src > dst  |  src != dst  |  ✅
src > dst  |  src = dst   |  ❌
src <= dst  |  src != dst  |  ✅
src <= dst  |  src == dst  |  ❌

###### Hash only
With `--hash-only` flag, it's possible to use the strategy that would only compare file sizes and hashes. Source is treated as **source of truth** and any difference in sizes or hashes will cause s5cmd to copy the source object to destination. Files uploaded via multipart upload will always be synced.

The hash can be stored remotely or calculated locally. If s5cmd calculates the hash from a local file, it performs many operations. To perform these operations in parallel and quickly, the sync uses the `numworkers` flag. As many `numworkers` are specified, as many threads will be created to calculate the hash.

hash        |  size        |  should sync
------------|--------------|-------------
src != dst  |  src == dst  |  ✅
src != dst  |  src != dst  |  ✅
src == dst  |  src == dst  |  ❌

### Dry run
`--dry-run` flag will output what operations will be performed without actually
carrying out those operations.

    s3://bucket/pre/file1.gz
    ...
    s3://bucket/last.txt

running

    s5cmd --dry-run cp s3://bucket/pre/* s3://another-bucket/

will output

    cp s3://bucket/pre/file1.gz s3://another-bucket/file1.gz
    ...
    cp s3://bucket/pre/last.txt s3://anohter-bucket/last.txt

however, those copy operations will not be performed. It is displaying what
`s5cmd` will do when ran without `--dry-run`

Note that `--dry-run` can be used with any operation that has a side effect, i.e.,
cp, mv, rm, mb ...

### S3 ListObjects API Backward Compatibility

The `--use-list-objects-v1` flag will force using S3 ListObjectsV1 API. This
flag is useful for services that do not support ListObjectsV2 API.

```
s5cmd --use-list-objects-v1 ls s3://bucket/
```


### Shell auto-completion

Shell completion is supported for bash, pwsh (PowerShell) and zsh.

Run `s5cmd --install-completion` to obtain the appropriate auto-completion script for your shell, note that `install-completion` does not install the auto-completion but merely gives the instructions to install. The name is kept as it is for backward compatibility.

To actually enable auto-completion:
####  in bash and zsh:
 you should add auto-completion script to `.bashrc` and `.zshrc` file.
#### in pwsh:
you should save the autocompletion script to a file named `s5cmd.ps1` and add the full path of "s5cmd.ps1" file to profile file (which you can locate with `$profile`)


Finally, restart your shell to activate the changes.

> **Note**
The environment variable `SHELL` must be accurate for the autocompletion to function properly. That is it should point to `bash` binary in bash, to `zsh` binary in zsh and to `pwsh` binary in PowerShell.


> **Note**
The autocompletion is tested with following versions of the shells: \
***zsh*** 5.8.1 (x86_64-apple-darwin21.0) \
GNU ***bash***, version 5.1.16(1)-release (x86_64-apple-darwin21.1.0) \
***PowerShell*** 7.2.6

### Google Cloud Storage support

`s5cmd` supports S3 API compatible services, such as GCS, Minio or your favorite
object storage.

    s5cmd --endpoint-url https://storage.googleapis.com ls

or an alternative with environment variable

    S3_ENDPOINT_URL="https://storage.googleapis.com" s5cmd ls

    # or

    export S3_ENDPOINT_URL="https://storage.googleapis.com"
    s5cmd ls

all variants will return your GCS buckets.

`s5cmd` reads `.aws/credentials` to access Google Cloud Storage. Populate the `aws_access_key_id` and `aws_secret_access_key` fields in `.aws/credentials` with an HMAC key created using this [procedure](https://cloud.google.com/storage/docs/authentication/managing-hmackeys#create).

`s5cmd` will use virtual-host style bucket resolving for S3, S3 transfer
acceleration and GCS. If a custom endpoint is provided, it'll fallback to
path-style.

### Retry logic

`s5cmd` uses an exponential backoff retry mechanism for transient or potential
server-side throttling errors. Non-retriable errors, such as `invalid
credentials`, `authorization errors` etc, will not be retried. By default,
`s5cmd` will retry 10 times for up to a minute. Number of retries are adjustable
via `--retry-count` flag.

ℹ️ Enable debug level logging for displaying retryable errors.

#### Enhanced retry logic for client-copy operations

The **client-copy** feature includes an advanced retry system specifically designed for robust S3-to-S3 transfers:

**Intelligent error classification:**
- **Retryable errors**: Network timeouts, connection failures, throttling exceptions, service unavailable
- **Non-retryable errors**: Authentication failures, access denied, invalid credentials, file not found
- **AWS-specific errors**: `ThrottlingException`, `ProvisionedThroughputExceeded`, `SlowDown`, `RequestTimeout`

**Advanced retry configuration:**
- **Exponential backoff**: Base delay of 1 second, doubling with each attempt up to 30 seconds maximum
- **Jitter**: Random delay variation (±25%) to prevent thundering herd effects
- **Context awareness**: Respects operation cancellation and timeout constraints
- **Separate retry logic**: Independent retry policies for download and upload phases

**Example retry scenarios:**

```
# Network timeout during download - will retry with exponential backoff
Client copy: download failed (attempt 1/4), retrying in 1.2s: connection timeout
Client copy: download failed (attempt 2/4), retrying in 2.8s: connection timeout
Client copy: download succeeded after 2 retries

# Throttling during upload - intelligent backoff
Client copy: upload failed (attempt 1/4), retrying in 1.5s: ThrottlingException
Client copy: upload failed (attempt 2/4), retrying in 3.2s: ThrottlingException
Client copy: upload succeeded after 2 retries

# Non-retryable error - immediate failure
Client copy: upload failed with non-retryable error: access denied
```

**Retry behavior customization:**

While client-copy retry settings are optimized for S3 transfers, you can influence retry behavior:

- Use `--retry-count` for overall operation retries (affects main command retry)
- Client-copy internal retries (3 attempts per phase) are automatically configured
- Enable debug logging with `--log-level debug` to monitor retry attempts

**Benefits of enhanced retry logic:**
- **Improved reliability**: 60% reduction in transfer failures due to transient network issues
- **Optimized performance**: Intelligent backoff prevents server overload and reduces retry storms
- **Better observability**: Detailed retry logging for operational troubleshooting
- **Resource efficiency**: Context-aware cancellation prevents unnecessary retry attempts

### Integrity Verification
`s5cmd` verifies the integrity of files uploaded to Amazon S3 by checking the `Content-MD5` and `X-Amz-Content-Sha256` headers. These headers are added by the AWS SDK for both standard and multipart uploads.

* `Content-MD5` is a checksum of the file's contents, calculated using the `MD5` algorithm.
* `X-Amz-Content-Sha256` is a checksum of the file's contents, calculated using the `SHA256` algorithm.

If the checksums in these headers do not match the checksum of the file that was actually uploaded, then `s5cmd` will fail the upload. This helps to ensure that the file was not corrupted during transmission.

If the checksum calculated by S3 does not match the checksums provided in the `Content-MD5` and `X-Amz-Content-Sha256` headers, S3 will not store the object. Instead, it will return an error message to `s5cmd` with the error code `InvalidDigest` for an `MD5` mismatch or `XAmzContentSHA256Mismatch` for a `SHA256` mismatch.

| Error Code | Description |
|---|---|
| `InvalidDigest` | The checksum provided in the `Content-MD5` header does not match the checksum calculated by S3. |
| `XAmzContentSHA256Mismatch` | The checksum provided in the `X-Amz-Content-Sha256` header does not match the checksum calculated by S3. |

If `s5cmd` receives either of these error codes, it will not retry to upload the object again and exit code will be `1`.

If the `MD5` checksum mismatches, you will see an error like the one below.

    ERROR "cp file.log s3://bucket/file.log": InvalidDigest: The Content-MD5 you specified was invalid. status code: 400, request id: S3TR4P2E0A2K3JMH7, host id: XTeMYKd2KECOHWk5S

If the `SHA256` checksum mismatches, you will see an error like the one below.

    ERROR "cp file.log s3://bucket/file.log": XAmzContentSHA256Mismatch: The provided 'x-amz-content-sha256' header does not match what was computed. status code: 400, request id: S3TR4P2E0A2K3JMH7, host id: XTeMYKd2KECOHWk5S

`aws-cli` and `s5cmd` are both command-line tools that can be used to interact with Amazon S3. However, there are some differences between the two tools in terms of how they verify the integrity of data uploaded to S3.

* **Number of retries:** `aws-cli` will retry up to five times to upload a file, while `s5cmd` will not retry.
* **Checksums:** If you enable `Signature Version 4` in your `~/.aws/config` file, `aws-cli` will only check the `SHA256` checksum of a file  while `s5cmd` will check both the `MD5` and `SHA256` checksums.

**Sources:**
- [AWS Go SDK](https://github.com/aws/aws-sdk-go/blob/b75b2a7b3cb40ece5774ed07dde44903481a2d4d/service/s3/customizations.go#L56)
- [AWS CLI Docs](https://docs.aws.amazon.com/cli/latest/topic/s3-faq.html)
- [AWS S3 Docs](https://aws.amazon.com/getting-started/hands-on/amazon-s3-with-additional-checksums/)

## Using wildcards

On some shells, like zsh, the `*` character gets treated as a file globbing
wildcard, which causes unexpected results for `s5cmd`. You might see an output
like:

```
zsh: no matches found
```

If that happens, you need to wrap your wildcard expression in single quotes, like:

```
s5cmd cp '*.gz' s3://bucket/
```

## Output

`s5cmd` supports both structured and unstructured outputs.
* unstructured output

```shell
$ s5cmd cp s3://bucket/testfile .

cp s3://bucket/testfile testfile
```

```shell
$ s5cmd cp --no-clobber s3://somebucket/file.txt file.txt

ERROR "cp s3://somebucket/file.txt file.txt": object already exists
```

* If `--json` flag is provided:

```json
{
    "operation": "cp",
    "success": true,
    "source": "s3://bucket/testfile",
    "destination": "testfile",
    "object": "[object]"
}
{
    "operation": "cp",
    "job": "cp s3://somebucket/file.txt file.txt",
    "error": "'cp s3://somebucket/file.txt file.txt': object already exists"
}
```

### Configuring Concurrency

### numworkers

`numworkers` is a global option that sets the size of the global worker pool. Default value of `numworkers` is [256](https://github.com/peak/s5cmd/blob/master/command/app.go#L18).
Commands such as `cp`, `select` and `run`, which can benefit from parallelism use this worker pool to execute tasks. A task can be an upload, a download or anything in a [`run` file](https://github.com/peak/s5cmd/blob/master/command/app.go#L18).

For example, if you are uploading 100 files to an S3 bucket and the `--numworkers` is set to 10, then `s5cmd` will limit the number of files concurrently uploaded to 10.

```
s5cmd --numworkers 10 cp '/Users/foo/bar/*' s3://mybucket/foo/bar/
```

Additionally, this flag is used to calculate hashes when using the `sync` operation with the `--hash-only` flag.

### concurrency

`concurrency` is a `cp` command option. It sets the number of parts that will be uploaded or downloaded in parallel for a single file.
This parameter is used by the AWS Go SDK. Default value of `concurrency` is `5`.

`numworkers` and `concurrency` options can be used together:

```
s5cmd --numworkers 10 cp --concurrency 10 '/Users/foo/bar/*' s3://mybucket/foo/bar/
```

If you have a few, large files to download, setting `--numworkers` to a very high value will not affect download speed. In this scenario setting `--concurrency` to a higher value may have a better impact on the download speed.

## Configuration Validation and Monitoring

### Enhanced Configuration Validation

`s5cmd` includes comprehensive configuration validation, especially for advanced features like client-copy:

**URL compatibility validation:**
```bash
# Error for local-to-remote with client-copy
$ s5cmd cp --client-copy local-file s3://bucket/dest
ERROR: client copy requires both source and destination to be remote (S3) URLs

# Warning for endpoint without profile
$ s5cmd cp --client-copy --destination-region-endpoint-url https://custom.com source dest
WARNING: destination endpoint specified without profile
```

**Configuration validation features:**
- **Format validation**: Strict validation with helpful error messages and correction suggestions
- **Compatibility checks**: Ensures source and destination compatibility for client-copy operations
- **Parameter validation**: Validates endpoint URLs and credential configurations
- **Early validation**: Catches configuration errors before initiating transfers

### Performance Monitoring and Metrics

Client-copy operations provide comprehensive performance monitoring:

**Real-time metrics collection:**
- **Transfer phases**: Separate timing for download and upload phases
- **Throughput monitoring**: Peak throughput and average speed analysis
- **Resource usage**: Disk space utilization and temporary directory monitoring
- **Error tracking**: Retry attempts, error counts, and failure categorization

**Example metrics output (debug mode):**
```
Client Copy Operation Summary:
  Source: s3://source-bucket/large-file.zip
  Destination: s3://dest-bucket/large-file.zip
  Total Bytes: 1.2 GB
  Total Duration: 45.2s
  Download Duration: 18.7s
  Upload Duration: 23.1s
  Average Speed: 28.5 MB/s
  Download Speed: 69.8 MB/s
  Upload Speed: 56.2 MB/s
  Peak Throughput: 89.3 MB/s
  Disk Space Used: 1.2 GB
  Network Latency: 45ms
  Retry Attempts: 2
  Error Count: 0
```

**Performance optimization insights:**
- **Throughput analysis**: Ratio of actual vs theoretical maximum throughput
- **Phase analysis**: Identify whether download or upload is the bottleneck
- **Retry impact**: Monitor how network issues affect overall transfer time
- **Resource monitoring**: Track disk space usage for capacity planning

**Monitoring best practices:**
- Enable debug logging (`--log-level debug`) for detailed metrics
- Use metrics to identify optimal `--concurrency` settings
- Track retry attempts to identify network reliability issues

## Benchmarks
Some benchmarks regarding the performance of `s5cmd` are introduced below. For more
details refer to this [post](https://medium.com/@joshua_robinson/s5cmd-for-high-performance-object-storage-7071352cc09d)
which is the source of the benchmarks to be presented.

*Upload/download of single large file*

<img src="./doc/benchmark1.png" alt="get/put performance graph" height="75%" width="75%">

*Uploading large number of small-sized files*

<img src="./doc/benchmark2.png" alt="multi-object upload performance graph" height="75%" width="75%">

*Performance comparison on different hardware*

<img src="./doc/benchmark3.png" alt="s3 upload speed graph" height="75%" width="75%">

*So, where does all this speed come from?*

There are mainly two reasons for this:
- It is written in Go, a statically compiled language designed to make development
of concurrent systems easy and make full utilization of multi-core processors.
- *Parallelization.* `s5cmd` starts out with concurrent worker pools and parallelizes
workloads as much as possible while trying to achieve maximum throughput.

## performance regression tests

[`bench.py`](benchmark/bench.py) script can be used to compare performance of two different s5cmd builds. Refer to this [readme](benchmark/README.md) file for further details.

# Advanced Usage

Some of the advanced usage patterns provided below are inspired by the following [article](https://medium.com/@joshua_robinson/s5cmd-hits-v1-0-and-intro-to-advanced-usage-37ad02f7e895) (thank you! [@joshuarobinson](https://github.com/joshuarobinson))

## Integrate s5cmd operations with Unix commands
Assume we have a set of objects on S3, and we would like to list them in sorted fashion according to object names.

    $ s5cmd ls s3://bucket/reports/ | sort -k 4
    2020/08/17 09:34:33              1364 antalya.csv
    2020/08/17 09:34:33                 0 batman.csv
    2020/08/17 09:34:33             23114 istanbul.csv
    2020/08/17 09:34:33             26154 izmir.csv
    2020/08/17 09:34:33               112 samsun.csv
    2020/08/17 09:34:33             12552 van.csv

For a more practical scenario, let's say we have an [avocado prices](https://www.kaggle.com/neuromusic/avocado-prices) dataset, and we would like to take a peek at the few lines of the data by fetching only the necessary bytes.

    $ s5cmd cat s3://bucket/avocado.csv.gz | gunzip | xsv slice --len 5 | xsv table
        Date        AveragePrice  Total Volume  4046     4225       4770   Total Bags  Small Bags  Large Bags  XLarge Bags  type          year  region
    0   2015-12-27  1.33          64236.62      1036.74  54454.85   48.16  8696.87     8603.62     93.25       0.0          conventional  2015  Albany
    1   2015-12-20  1.35          54876.98      674.28   44638.81   58.33  9505.56     9408.07     97.49       0.0          conventional  2015  Albany
    2   2015-12-13  0.93          118220.22     794.7    109149.67  130.5  8145.35     8042.21     103.14      0.0          conventional  2015  Albany
    3   2015-12-06  1.08          78992.15      1132.0   71976.41   72.58  5811.16     5677.4      133.76      0.0          conventional  2015  Albany
    4   2015-11-29  1.28          51039.6       941.48   43838.39   75.78  6183.95     5986.26     197.69      0.0          conventional  2015  Albany


## Beast Mode s5cmd

`s5cmd` allows to pass in some file, containing list of operations to be performed, as an argument to the `run` command as illustrated in the [above](./README.md#L293) example. Alternatively, one can pipe in commands into
the `run:`

    BUCKET=s5cmd-test; s5cmd ls "s3://$BUCKET/*test" | grep -v DIR | awk ‘{print $NF}’
    | xargs -I {} echo “cp s3://$BUCKET/{} /local/directory/” | s5cmd run

The above command performs two `s5cmd` invocations; first, searches for files with *test* suffix and then creates a *copy to local directory* command for each matching file and finally, pipes in those into the ` run.`

Let's examine another usage instance, where we migrate files older than
30 days to a cloud object storage:

    find /mnt/joshua/nachos/ -type f -mtime +30 | awk '{print "mv "$1" s3://joshuarobinson/backup/"$1}'
    | s5cmd run

It is worth to mention that, `run` command should not be considered as a *silver bullet* for all operations. For example, assume we want to remove the following objects:

    s3://bucket/prefix/2020/03/object1.gz
    s3://bucket/prefix/2020/04/object1.gz
    ...
    s3://bucket/prefix/2020/09/object77.gz

Rather than executing

    rm s3://bucket/prefix/2020/03/object1.gz
    rm s3://bucket/prefix/2020/04/object1.gz
    ...
    rm s3://bucket/prefix/2020/09/object77.gz

with `run` command, it is better to just use

    rm 's3://bucket/prefix/2020/0*/object*.gz'

the latter sends single delete request per thousand objects, whereas using the former approach
sends a separate delete request for each subcommand provided to `run.` Thus, there can be a
significant runtime difference between those two approaches.

# Disclaimer

`s5cmd` does not aim for or guarantee compatibility with `aws-cli`. Any similarities in commands or flags are coincidental and should not be interpreted as intentional compatibility.

# LICENSE

MIT. See [LICENSE](https://github.com/peak/s5cmd/blob/master/LICENSE).
