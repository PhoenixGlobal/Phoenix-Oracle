package utils

import (
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"PhoenixOracle/lib/logger"

	"github.com/pkg/errors"
)

func FileExists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}
	return true
}

func TooPermissive(fileMode, maxAllowedPerms os.FileMode) bool {
	return fileMode&^maxAllowedPerms != 0
}

func IsFileOwnedByPhoenix(fileInfo os.FileInfo) (bool, error) {
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, errors.Errorf("Unable to determine file owner of %s", fileInfo.Name())
	}
	return int(stat.Uid) == os.Getuid(), nil
}

func EnsureDirAndMaxPerms(path string, perms os.FileMode) error {
	stat, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		// Regular error
		return err
	} else if os.IsNotExist(err) {
		// Dir doesn't exist, create it with desired perms
		return os.MkdirAll(path, perms)
	} else if !stat.IsDir() {
		// Path exists, but it's a file, so don't clobber
		return errors.Errorf("%v already exists and is not a directory", path)
	} else if stat.Mode() != perms {
		// Dir exists, but wrong perms, so chmod
		return os.Chmod(path, (stat.Mode() & perms))
	}
	return nil
}

func WriteFileWithMaxPerms(path string, data []byte, perms os.FileMode) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	defer logger.ErrorIfCalling(f.Close)
	err = EnsureFileMaxPerms(f, perms)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func CopyFileWithMaxPerms(srcPath, dstPath string, perms os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrap(err, "could not open source file")
	}
	defer logger.ErrorIfCalling(src.Close)

	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return errors.Wrap(err, "could not open destination file")
	}
	defer logger.ErrorIfCalling(dst.Close)

	err = EnsureFileMaxPerms(dst, perms)
	if err != nil {
		return errors.Wrap(err, "could not set file permissions")
	}

	_, err = io.Copy(dst, src)
	return errors.Wrap(err, "could not copy file contents")
}

func EnsureFileMaxPerms(file *os.File, perms os.FileMode) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if stat.Mode() == perms {
		return nil
	}
	return file.Chmod(stat.Mode() & perms)
}

func EnsureFilepathMaxPerms(filepath string, perms os.FileMode) error {
	dst, err := os.OpenFile(filepath, os.O_RDWR, perms)
	if err != nil {
		return err
	}
	defer logger.ErrorIfCalling(dst.Close)

	return EnsureFileMaxPerms(dst, perms)
}

func FilesInDir(dir string) ([]string, error) {
	f, err := os.Open(dir)
	if err != nil {
		return []string{}, err
	}
	defer logger.ErrorIfCalling(f.Close)

	r, err := f.Readdirnames(-1)
	if err != nil {
		return []string{}, err
	}

	return r, nil
}

func FileContents(path string) (string, error) {
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}
