package tests

import (
	"path/filepath"
)

/*
RepoAbsPath returns the absolute path of the repository. If a different package
uses this, this function may return the wrong result. This is based on verbiage
in "go help testflag":

	When 'go test' runs a test binary, it does so from within the
	corresponding package's source code directory.
*/
func RepoAbsPath() (string, error) {
	// This file is one-deep, so we can return a directory that is one up
	// from this file. This is immune from assumed values of os.Getwd().
	return filepath.Abs("..")
}

// DistAbsPath returns the absolute path of the dist/ subdirectory.
func DistAbsPath() (string, error) {
	repo, err := RepoAbsPath()
	if err != nil {
		return repo, err
	}
	return filepath.Join(repo, "dist"), nil
}

// ZQAbsPath returns the absolute path of the zq binary in dist/.
func ZQAbsPath() (string, error) {
	distdir, err := DistAbsPath()
	if err != nil {
		return distdir, err
	}
	return filepath.Join(distdir, "zq"), nil
}

// ZQSampleDataAbsPath returns the absolute path of zed-sample-data.
func ZQSampleDataAbsPath() (string, error) {
	repo, err := RepoAbsPath()
	if err != nil {
		return repo, err
	}
	return filepath.Join(repo, "zed-sample-data"), nil
}
