package repodb_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/readpe/repodb"
)

// FileRecord is a simple file record, satisfying the Record interface
type FileRecord struct {
	Name        string    `json:"name"`
	SoftDeleted bool      `json:"softdeleted"`
	CreateOn    time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	DeletedOn   time.Time `json:"deleted_on"`
	body        string    // unexported so as not to add body to meta-data in addition to file
}

// FileName returns the file name for the record. Satisfies Record interface
func (fr *FileRecord) FileName() string {
	return fr.Name
}

// Folder returns the folder name for the record within the repository. Satisfies Record interface
func (fr *FileRecord) Folder() string {
	return "files"
}

func Example() {
	// temp directory used for example
	dir, err := ioutil.TempDir(os.TempDir(), "repodb")
	if err != nil {
		log.Fatal(err)
	}

	// create database and repository
	db := repodb.NewDB(dir)

	// setup Repo details for CreateRepo
	repo := &repodb.Repo{
		Name:        "HelloRepo",
		DB:          db,
		Description: "A hello world repository.",
		Protected:   false,
		SoftDeleted: false,
		CreatedOn:   time.Now(),
		UpdatedOn:   time.Now(),
	}

	// create repository: adds a directory does git init, and writes meta-data for repo
	err = db.CreateRepo(repo)
	if err != nil {
		log.Fatal(err)
	}

	// creating example record
	fr := &FileRecord{
		Name:        "HelloWorld.txt",
		SoftDeleted: false,
		CreateOn:    time.Now(),
		UpdatedOn:   time.Now(),
		body:        "This is the body of the text file, an io.Reader could be used instead.",
	}

	// re-open repository for each write/read
	repo, err = db.OpenRepo("HelloRepo")
	if err != nil {
		log.Fatal(err)
	}

	// writes file to repository and commits all changes
	repo.WriteFile(fr, strings.NewReader(fr.body), repodb.CommitOptions{
		Msg: fmt.Sprintf("added file %s to %s", fr.FileName(), repo.Dir()),
	})
	repo.WriteMeta(fr, repodb.CommitOptions{
		Msg: fmt.Sprintf("added meta-data for file %s to %s", fr.FileName(), repo.Dir()),
	})

	// Read File
	buf := bytes.NewBufferString("")
	_, err = repo.ReadFile(fr, buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(buf)
	// Output: This is the body of the text file, an io.Reader could be used instead.
}
