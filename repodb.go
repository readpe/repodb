package repodb

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	scribble "github.com/nanobox-io/golang-scribble"
)

// package variables
var (
	MetaDir             = "meta-data"
	DBRepoName          = "db-repo" // the database repo name
	DBRepoCommitOptions = CommitOptions{
		Msg: DBRepoName,
		Opts: git.CommitOptions{
			Author:    &object.Signature{Name: "repodb", Email: ""},
			Committer: &object.Signature{Name: "repodb", Email: ""},
		},
	}
)

// errors
var (
	ErrRepoAlreadyExists = errors.New("repo already exists")
	ErrRepoNotExists     = errors.New("repo does not exist")
)

// CommitOptions is a wrapper struct arount git.CommitOptions with the addition of the message
type CommitOptions struct {
	Msg  string
	Opts git.CommitOptions
}

// Record is a RepoDB record interface.
type Record interface {
	FileName() string
	Folder() string
}

// RepoDB is a file based database of git repositories.
type RepoDB struct {
	sync.RWMutex
	dir string
}

// NewDB returns a new RepoDB in the named directory
func NewDB(dir string) *RepoDB {

	db := &RepoDB{
		dir: dir,
	}
	return db
}

// CreateRepo will create a git repository as a subdirectory dir in the RepoDB.
// Will return ErrRepoAlreadyExists if it already exists
func (db *RepoDB) CreateRepo(repo *Repo) error {
	db.Lock()
	defer db.Unlock()

	if repo == nil {
		return fmt.Errorf("CreateRepo repo pointer cannot be nil")
	}

	// don't allow .. or Pathseparator in repo Name
	repo.Name = cleanPath(repo.Name)
	if repo.Name == "" {
		return fmt.Errorf("CreateRepo repo name cannot be empty")
	}

	_, err := git.PlainInit(repo.Dir(), false)
	switch {
	case errors.Is(err, git.ErrRepositoryAlreadyExists):
		return ErrRepoAlreadyExists
	case err != nil:
		return fmt.Errorf("unable to create repo at %s: %v", repo.Dir(), err)
	}
	err = repo.WriteMeta(repo, DBRepoCommitOptions)
	if err != nil {
		return err
	}

	return nil
}

// OpenRepo will open the git repository at the specified directory Will return ErrRepoNotExists if no valid repository is found
func (db *RepoDB) OpenRepo(name string) (*Repo, error) {
	db.Lock()
	defer db.Unlock()

	// don't allow .. or Pathseparator in repo Name
	name = cleanPath(name)

	repo := &Repo{
		Name: name,
		DB:   db,
	}

	_, err := git.PlainOpen(repo.Dir())
	switch {
	case errors.Is(err, git.ErrRepositoryNotExists):
		return nil, ErrRepoNotExists
	case err != nil:
		return nil, fmt.Errorf("unable to open repo at %s: %v", repo.Dir(), err)
	}

	err = repo.LoadMeta(repo)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// RemoveRepo will remove the current database and all files/sub-directories. Use with caution.
func (db *RepoDB) RemoveRepo(dir string) error {

	// don't allow .. or Pathseparator in repo Name
	dir = cleanPath(dir)

	repo, err := db.OpenRepo(dir)

	switch {
	case errors.Is(err, ErrRepoAlreadyExists):
		// okay to have exists error for this method
	case err != nil:
		return fmt.Errorf("unable to remove repo %s: %v", dir, err)
	}
	db.Lock()
	defer db.Unlock()
	return os.RemoveAll(repo.Dir())
}

// ListRepos returns a list of repositories in the database
func (db *RepoDB) ListRepos() []*Repo {
	repos := []*Repo{}

	fileInfos, err := ioutil.ReadDir(db.dir)
	if err != nil {
		return repos
	}
	for _, f := range fileInfos {
		repo, err := db.OpenRepo(f.Name())
		if err != nil {
			continue
		}
		repos = append(repos, repo)
	}
	return repos
}

// Repo is a git repository as a subdirectory under the RepoDB
type Repo struct {
	sync.RWMutex
	Name        string
	DB          *RepoDB `json:"-"`
	Description string
	Protected   bool
	SoftDeleted bool
	CreatedOn   time.Time
	UpdatedOn   time.Time
	DeletedOn   time.Time
}

// Protect the repo from deletion
func (repo *Repo) Protect() error {
	repo.Protected = true
	err := repo.WriteMeta(repo, DBRepoCommitOptions)
	if err != nil {
		return fmt.Errorf("unable to protect repo %s", repo.Dir())
	}
	return nil
}

// Dir is the full directory for the Repo under the DB
func (repo *Repo) Dir() string {
	return path.Clean(path.Join(repo.DB.dir, repo.Name))
}

// FileName returns the repo , which is its dir implements Record interface
func (repo *Repo) FileName() string {
	return repo.Name
}

// Folder is the record folder, final path component under the Repo. Implements Record interface
func (repo *Repo) Folder() string {
	return "repos"
}

// CommitAll does a git add . && git commit -m "msg"
func (repo *Repo) CommitAll(opts CommitOptions) error {
	r, err := git.PlainOpen(repo.Dir())
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	_, err = w.Add(".")
	if err != nil {
		return err
	}
	s, _ := w.Status()
	if s.IsClean() {
		return nil
	}

	// remove leading and trailing spaces from message
	opts.Msg = strings.TrimSpace(opts.Msg)

	// sets When for both Author and Commiter to time.Now
	if opts.Opts.Author != nil {
		opts.Opts.Author.When = time.Now()
	}
	if opts.Opts.Committer != nil {
		opts.Opts.Committer.When = time.Now()
	}

	_, err = w.Commit(opts.Msg, &opts.Opts)
	if err != nil {
		return err
	}
	return nil
}

// FileExists checks if file exists
func (repo *Repo) FileExists(rec Record) bool {
	filename := path.Join(repo.Dir(), rec.Folder(), rec.FileName())
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// WriteFile will create and write the record to file. If the directory does not exist, it will be created.
func (repo *Repo) WriteFile(rec Record, r io.Reader, opts CommitOptions) error {
	// reader is nil, return
	if r == nil {
		return fmt.Errorf("WriteFile requires non-nil reader: %s", rec.FileName())
	}
	repo.Lock()
	defer repo.Unlock()

	dir := path.Join(repo.Dir(), rec.Folder())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("unable to make directory %s: %v", dir, err)
	}

	f, err := os.Create(path.Join(dir, rec.FileName()))
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", rec.FileName(), err)
	}

	// Copy from the record reader to the created file.
	n, err := io.Copy(f, r)
	if err != nil {
		return fmt.Errorf("copy failed to %s: %v", rec.FileName(), err)
	}

	// appends to git commit message separated by blank line. If original message is blank it will remove leading blank spaces
	opts.Msg = fmt.Sprintf("%s\n\nwrote %d bytes to file %s", opts.Msg, n, path.Join(rec.Folder(), rec.FileName()))

	return repo.CommitAll(opts)
}

// ReadFile will read the file to the provided io.Writer
func (repo *Repo) ReadFile(rec Record, w io.Writer) (written int64, err error) {
	// reader is nil, return
	if w == nil {
		return 0, fmt.Errorf("ReadFile requires non-nil writer: %s", rec.FileName())
	}

	repo.RLock()
	defer repo.RUnlock()

	filename := path.Join(repo.Dir(), rec.Folder(), rec.FileName())
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(w, f)
	return n, err
}

// RemoveFile removes the record. If there is an error it will
// be of type *os.PathError. This function will not remove the
// coresponding meta-data file, use in conjunction with RemoveMeta.
func (repo *Repo) RemoveFile(rec Record, opts CommitOptions) error {
	repo.Lock()
	defer repo.Unlock()

	filename := path.Join(repo.Dir(), rec.Folder(), rec.FileName())
	err := os.Remove(filename)
	if err != nil {
		return err
	}

	// appends to git commit message separated by blank line. If original message is blank it will remove leading blank spaces
	opts.Msg = fmt.Sprintf("%s\n\nremoved file %s", opts.Msg, filename)

	return repo.CommitAll(opts)
}

// WriteMeta data for record to json file db.
func (repo *Repo) WriteMeta(rec Record, opts CommitOptions) error {
	repo.Lock()
	defer repo.Unlock()
	dir := path.Join(repo.Dir(), rec.Folder())
	_, ok := rec.(*Repo)
	if ok {
		dir = path.Join(repo.Dir(), "")
	}

	// TODO(readpe): create own scribble package without logger
	meta, err := scribble.New(dir, &scribble.Options{})
	if err != nil {
		return fmt.Errorf("cannot create scribble db %s: %v", dir, err)
	}

	err = meta.Write(MetaDir, rec.FileName(), rec)
	if err != nil {
		return fmt.Errorf("cannot write meta-data for %s: %v", rec.FileName(), err)
	}

	// appends to git commit message separated by blank line. If original message is blank it will remove leading blank spaces
	opts.Msg = fmt.Sprintf("%s\n\nwrote meta-data to %s", opts.Msg, path.Join(MetaDir, rec.FileName())+".json")

	return repo.CommitAll(opts)
}

// LoadMeta data for record to Record concrete type
func (repo *Repo) LoadMeta(rec Record) error {
	repo.RLock()
	repo.RUnlock()

	dir := path.Join(repo.Dir(), rec.Folder())
	if _, ok := rec.(*Repo); ok {
		dir = path.Join(repo.Dir())
	}

	meta, err := scribble.New(dir, &scribble.Options{})
	if err != nil {
		return fmt.Errorf("cannot load scribble db %s: %v", dir, err)
	}

	err = meta.Read(MetaDir, rec.FileName(), rec)
	if err != nil {
		return fmt.Errorf("cannot write meta-data for %s: %v", rec.FileName(), err)
	}
	return nil
}

// RemoveMeta removes the records meta-data file. If there is an error it will
// be of type *os.PathError. This function will not remove the
// referenced record file, use in conjunction with RemoveFIle.
func (repo *Repo) RemoveMeta(rec Record, opts CommitOptions) error {
	repo.Lock()
	defer repo.Unlock()

	filename := path.Join(repo.Dir(), rec.Folder(), MetaDir, rec.FileName()) + ".json"
	err := os.Remove(filename)
	if err != nil {
		return err
	}

	// appends to git commit message separated by blank line. If original message is blank it will remove leading blank spaces
	opts.Msg = fmt.Sprintf("%s\n\nremoved meta-data file %s", opts.Msg, filename)

	return repo.CommitAll(opts)
}

// cleanPath used to remove .. and PathSeparator from file and directory names
func cleanPath(s string) string {
	s = strings.ReplaceAll(s, "..", "")
	return strings.ReplaceAll(s, string(os.PathSeparator), "")
}
