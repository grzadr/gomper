package scanner

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

type failingProfilesFS struct{}

func (f failingProfilesFS) Open(name string) (fs.File, error) {
	return nil, errors.New("simulated fs open error")
}

func (f failingProfilesFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, errors.New("simulated readdir error")
}

func (f failingProfilesFS) ReadFile(name string) ([]byte, error) {
	return nil, errors.New("simulated readfile error")
}

func TestListProfiles_ReadDirError(t *testing.T) {
	origFS := profilesFS
	profilesFS = failingProfilesFS{}
	defer func() { profilesFS = origFS }()

	profiles, err := ListProfiles()
	if err == nil {
		t.Fatalf("expected error from ListProfiles when ReadDir fails, got nil")
	}
	if profiles != nil {
		t.Errorf("expected nil profiles slice on error, got: %v", profiles)
	}
	if !strings.Contains(err.Error(), "failed to read embedded profiles directory") {
		t.Errorf("expected error message mentioning failure to read embedded profiles directory, got: %v", err)
	}
}
