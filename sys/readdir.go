// Package sys provides system level utilities.
package sys

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// La lists all files of the directory including
// hidden files and self/parent directory;
// it's behaviour is the same as calling `ls -a`.
func La(name string) ([]string, error) {
	// TODO: cross platform
	path, err := exec.LookPath("ls")
	if err != nil {
		return nil, err
	}

	// run and get the output
	out, err := exec.Command(path, "-a", name).Output()
	if err != nil {
		return nil, err
	}

	ss := strings.Split(string(out), "\n")
	if ss[len(ss)-1] == "" {
		// trailing space
		return ss[:len(ss)-1], nil
	}

	return ss, nil
}

// ReadDir mimic the behaviour of ioutil.ReadDir
// (reads the directory named by dirname and returns
// a list of directory entries sorted by filename),
// but use `ls -a` and string parsing internally.
func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	f.Close()

	names, err := La(dirname)
	if err != nil {
		return nil, err
	}

	var list []os.FileInfo
	for _, name := range names {
		if name == "." || name == ".." {
			// ioutil.ReadDir ignores these
			continue
		}
		filename := filepath.Join(dirname, name)
		info, err := os.Lstat(filename)
		if err != nil {
			return list, err
		}
		list = append(list, info)
	}

	return list, nil
}

// from path.go

// walk recursively descends path, calling w.
func walk(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(path, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	// monkey-patching the read directory function
	// names, err := readDirNames(path)
	names, err := La(path)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, name := range names {
		if name == "." || name == ".." {
			// ignore self and parent dir
			continue
		}
		filename := filepath.Join(path, name)
		fileInfo, err := os.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func Walk(root string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		err = walkFn(root, nil, err)
	} else {
		err = walk(root, info, walkFn)
	}
	if err == filepath.SkipDir {
		return nil
	}
	return err
}
