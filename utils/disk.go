package utils

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"encoding/binary"
	"os"
	"os/user"
	"strconv"
	"strings"
	"fmt"
	"github.com/dunkelstern/libxhyve"
)

func changePermissions(path string) error {
	var uid, gid int
	var err error

	suid := os.Getenv("SUDO_UID")
	if suid != "" {
		uid, err = strconv.Atoi(suid)
		if err != nil {
			return err
		}
	} else {
		uid = os.Getuid()
	}

	sgid := os.Getenv("SUDO_GID")
	if sgid != "" {
		gid, err = strconv.Atoi(sgid)
		if err != nil {
			return err
		}
	} else {
		gid = os.Getgid()
	}

	return os.Chown(path, uid, gid)
}

func CreateDir() error {
	path := os.ExpandEnv("$HOME/.dlite")

	err := os.MkdirAll(path, 0755)
	if err != nil {
		return err
	}

	return changePermissions(path)
}

func RemoveDir() error {
	path := os.ExpandEnv("$HOME/.dlite")
	return os.RemoveAll(path)
}

func CreateDisk(sshKey string, size int) error {
	if strings.Contains(sshKey, "$HOME") {
		username := os.Getenv("SUDO_USER")
		if username == "" {
			username = os.Getenv("USER")
		}

		me, err := user.Lookup(username)
		if err != nil {
			return err
		}

		sshKey = strings.Replace(sshKey, "$HOME", me.HomeDir, -1)
	}

	// fetch ssh key and initialize tar file for setup mechanism
	sshKey = os.ExpandEnv(sshKey)
	keyBytes, err := ioutil.ReadFile(sshKey)
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	tarball := tar.NewWriter(buffer)
	files := []struct {
		Name string
		Body []byte
	}{
		{"dhyve, please format-me", []byte("dhyve, please format-me")},
		{".ssh/authorized_keys", keyBytes},
	}

	for _, file := range files {
		if err = tarball.WriteHeader(&tar.Header{
			Name: file.Name,
			Mode: 0644,
			Size: int64(len(file.Body)),
		}); err != nil {
			return err
		}

		if _, err = tarball.Write(file.Body); err != nil {
			return err
		}
	}

	if err = tarball.Close(); err != nil {
		return err
	}

	// initialize disk
	diskConfig := fmt.Sprintf("%s,sectorsize=4096,size=%dG,split=1G,sparse,reset", os.ExpandEnv("$HOME/.dlite/disk.img"), size)
	xhyve.InitDisk(diskConfig)

	// write tar file to first disk image
	path := os.ExpandEnv("$HOME/.dlite/disk.img.0000")
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()
	_, err = f.Write(buffer.Bytes())
	if err != nil {
		return err
	}

	// fill to next sector boundary
	for i := 0; i < (4096 - (len(buffer.Bytes()) % 4096)); i++ {
		err := binary.Write(f, binary.LittleEndian, int8(0))
		if err != nil {
			return err
		}
	}

	// update disk lut
	path = os.ExpandEnv("$HOME/.dlite/disk.img.lut")
	f, err = os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	defer f.Close()
	for i := 0; i < len(buffer.Bytes()); i+= 4096 {
		err := binary.Write(f, binary.LittleEndian, int32(i / 4096))
		if err != nil {
			return err
		}
	}

	return nil
}
