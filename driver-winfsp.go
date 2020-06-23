// +build windows

package main

// --------------------------------
// THIS FILE IS NOT QUITE COMPLETE!
// WINDOWS IS NOT SUPPORTED (YET)!
// (But the dev always forgets to
// skip this file when checking in,
// so this note avoids any confusion
// with you. Maybe. :-))
// --------------------------------

// Requires WinFsp.

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"sync"

	"github.com/billziss-gh/cgofuse/fuse"
)

// -------------------------
// WinFsp implementation:
// -------------------------

func (file *IndexedFile) setAttributes(out *fuse.Stat_t) {
	// Sets the file attributes.
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	out.Ctim = fuse.NewTimespec(file.Timestamp)
	out.Mtim = fuse.NewTimespec(file.Timestamp)

	out.Mode = file.mode()
	out.Ino = file.Inode

	out.Size = file.size()

	// Get the user's UID and GID and set them.
	uid32, _ := strconv.ParseUint(user.Uid, 10, 32)
	gid32, _ := strconv.ParseUint(user.Gid, 10, 32)
	out.Uid = uint32(uid32)
	out.Gid = uint32(gid32)
}

func (file *IndexedFile) size() int64 {
	// Returns the file size or 0 for directories.
	if file.IsDirectory == true {
		return 0
	} else {
		return int64(len(file.Data))
	}
}

func (file *IndexedFile) mode() uint32 {
	// Returns the mode depending on the type.
	if file.IsDirectory == true {
		return 0755 | fuse.S_IFDIR
	} else {
		return 0644 | fuse.S_IFREG
	}
}

type RssfsNode struct {
	stat fuse.Stat_t
	xatr map[string][]byte
	chld map[string]*RssfsNode
	data []byte
}

type Rssfs struct {
	fuse.FileSystemBase
	lock    sync.Mutex
	ino     uint64
	root    *RssfsNode
	openmap map[uint64]*RssfsNode
}

// tbd
// ?? http://www.secfs.net/winfsp/rel/
// ?? https://github.com/billziss-gh/cgofuse/blob/master/examples/memfs/memfs.go

func Mount(cfg RssfsConfig) {
	// Mounts the file system as instructed by the user.
	rssfs := &Rssfs{}
	host := fuse.NewFileSystemHost(rssfs)
	host.Mount(cfg.DriveLetter, os.Args[1:])

	fmt.Println("Ready! Unmount to terminate.")
}
