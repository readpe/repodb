# RepoDB
> RepoDB is a simple file based database of git repositories. Metadata for files and repositories is provided as flat json files within.

This module is used to facilitate a local file storage database with "automatic" version control support. The design focused on low-traffic applications with increased file auditing and retention. Performance may suffer for high traffic applications, although that has not been investigated.

## Installation
```sh
go get github.com/readpe/repodb
```

## Usage Example
**See [examples_test.go](examples_test.go) for full example.**

General steps:
1. Create file record type(s) satisfying Record interface
2. Create Database using NewDB
3. Create Repository using CreateRepo
4. Write/Read/Delete files in Repository

## License
Distributed under the MIT license. For more information, as well as third party licenses and notices, see ``LICENSE``.

## Acknowledgements

* [go-git](https://github.com/go-git/go-git)
* [golang-scribble](https://github.com/nanobox-io/golang-scribble)

