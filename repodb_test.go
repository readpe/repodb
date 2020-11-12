package repodb_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/readpe/repodb"
)

var (
	db *repodb.RepoDB
)

func TestMain(m *testing.M) {
	setup()
	m.Run()
}

func setup() string {
	dir, err := ioutil.TempDir(os.TempDir(), "repodb")
	if err != nil {
		panic(err)
	}
	db = repodb.NewDB(dir)
	return dir
}

func TestRepoDB_CreateRepo(t *testing.T) {
	type args struct {
		repo *repodb.Repo
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				&repodb.Repo{
					Name: "TestRepo",
					DB:   db,
				},
			},
			wantErr: false,
		},
		{
			name: "normal",
			args: args{
				&repodb.Repo{
					Name: "TestRepo1",
					DB:   db,
				},
			},
			wantErr: false,
		},
		{
			name: "normal",
			args: args{
				&repodb.Repo{
					Name: "TestRepo2",
					DB:   db,
				},
			},
			wantErr: false,
		},
		{
			name: "nil",
			args: args{
				nil,
			},
			wantErr: true,
		},
		{
			name: "empty name",
			args: args{
				&repodb.Repo{
					Name: "",
					DB:   db,
				},
			},
			wantErr: true,
		},
		{
			name: "..",
			args: args{
				&repodb.Repo{
					Name: "..",
					DB:   db,
				},
			},
			wantErr: true,
		},
		{
			name: "PathSep",
			args: args{
				&repodb.Repo{
					Name: string(os.PathSeparator),
					DB:   db,
				},
			},
			wantErr: true,
		},
		{
			name: "exists",
			args: args{
				&repodb.Repo{
					Name: "TestRepo",
					DB:   db,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.CreateRepo(tt.args.repo); (err != nil) != tt.wantErr {
				t.Errorf("RepoDB.CreateRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepoDB_OpenRepo(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "normal",
			args:    args{"TestRepo"},
			wantErr: false,
		},
		{
			name:    "empty",
			args:    args{""},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.OpenRepo(tt.args.name)
			if got == nil && err == nil {
				t.Errorf("RepoDB.OpenRepo() got nil repo, and err != nil")
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("RepoDB.OpenRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestRepoDB_ListRepos(t *testing.T) {
	want := 3
	got := len(db.ListRepos())
	if got != want {
		t.Errorf("RepoDB.ListRepos() = %v, want %v", got, want)
	}
}

func TestRepoDB_RemoveRepo(t *testing.T) {
	type args struct {
		dir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				"TestRepo",
			},
			wantErr: false,
		},
		{
			name: "repeated",
			args: args{
				"TestRepo",
			},
			wantErr: true,
		},
		{
			name: "empty",
			args: args{
				"",
			},
			wantErr: true,
		},
		{
			name: "..",
			args: args{
				"..",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := db.RemoveRepo(tt.args.dir); (err != nil) != tt.wantErr {
				t.Errorf("RepoDB.RemoveRepo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRepo_Protect(t *testing.T) {
	repo, err := db.OpenRepo("TestRepo2")
	if err != nil {
		t.Fatal(err)
	}

	err = repo.Protect()
	if err != nil {
		t.Fatal(err)
	}

	newRepo, err := db.OpenRepo("TestRepo2")
	if err != nil {
		t.Fatal(err)
	}
	if newRepo.Protected != true {
		t.Errorf("Repo.Protect() Protected = %v, %v", newRepo.Protected, true)
	}
}
