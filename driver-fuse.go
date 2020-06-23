// +build linux solaris dragonfly freebsd netbsd openbsd darwin

package main

import (
	"context"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/kyokomi/emoji"
)

var (
	config RssfsConfig
)

// -------------------------
// Go-FUSE implementation:
// -------------------------

func (file *IndexedFile) setAttributes(out *fuse.Attr) {
	// Sets the file attributes.
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	out.SetTimes(&file.Timestamp, &file.Timestamp, &file.Timestamp)

	out.Mode = file.mode()
	out.Ino = file.Inode

	out.Size = file.size()

	// Get the user's UID and GID and set them.
	uid32, _ := strconv.ParseUint(user.Uid, 10, 32)
	gid32, _ := strconv.ParseUint(user.Gid, 10, 32)
	out.Uid = uint32(uid32)
	out.Gid = uint32(gid32)
}

func (file *IndexedFile) size() uint64 {
	// Returns the file size or 0 for directories.
	if file.IsDirectory == true {
		return 0
	} else {
		return uint64(len(file.Data))
	}
}

func (file *IndexedFile) mode() uint32 {
	// Returns the mode depending on the type.
	if file.IsDirectory == true {
		return 0755 | uint32(syscall.S_IFDIR)
	} else {
		return 0644 | uint32(syscall.S_IFREG)
	}
}

type RssfsNode struct {
	fs.Inode
	path string
	data []byte
}

func NewRssfsNode(path string) fs.InodeEmbedder {
	// A constructor, mainly.
	return &RssfsNode{
		path: path,
	}
}

func (n *RssfsNode) currentPath() string {
	path := n.Path(nil)

	root := n.Root().Operations().(*RssfsNode)
	return filepath.Join(root.path, path)
}

func (n *RssfsNode) Opendir(ctx context.Context) syscall.Errno {
	// Checks whether the given category exists.
	path := n.currentPath()

	if _, found := tree[path]; found == false {
		return syscall.ENOENT
	}

	return fs.OK
}

func (n *RssfsNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	// Returns a list of file entries for currentPath().
	path := n.currentPath()

	if fileIndex[path].Feed != nil {
		// Feed objects are set for single feeds. The user seems to read
		// a single feed. Let's update it first.
		emoji.Println(":arrows_counterclockwise: Updating feed contents.")

		// TODO: We should probably call UpdateSingleFeed() here instead.
		//       But that breaks Lookup() yet as an already existing node
		//       cannot be created again. Investigate.
		// tree[path], _, _ = UpdateSingleFeed(fileIndex[path].Feed, fileIndex[path].Inode)

		tree = PopulateFeedTree(config)

		for parentPath, children := range tree {
			for _, child := range children {
				fullPath := filepath.Join(parentPath, child.Filename)
				fileIndex[fullPath] = child
			}
		}
	}

	files, found := tree[path]
	if found == false {
		return nil, syscall.ENOENT
	}

	entries := make([]fuse.DirEntry, len(files))
	for i, file := range files {
		entries[i] = fuse.DirEntry{
			Name: file.Filename,
			Mode: file.mode(),
			Ino:  file.Inode,
		}
	}

	ds := fs.NewListDirStream(entries)

	return ds, fs.OK
}

func (n *RssfsNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (childNode *fs.Inode, errno syscall.Errno) {
	// Generates a node for the given file.
	childPath := filepath.Join(n.currentPath(), name)

	entry, found := fileIndex[childPath]
	if found == false {
		return nil, syscall.ENOENT
	}

	entry.setAttributes(&out.Attr)

	childRssfsNode := NewRssfsNode(childPath)

	sa := fs.StableAttr{
		Mode: entry.mode(),
		Gen:  1,
		Ino:  entry.Inode,
	}

	childNode = n.NewInode(ctx, childRssfsNode, sa)

	return childNode, fs.OK
}

func (n *RssfsNode) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	path := n.currentPath()

	entry, found := fileIndex[path]
	if found == false {
		return syscall.ENOENT
	}

	entry.setAttributes(&out.Attr)

	return fs.OK
}

func (n *RssfsNode) Read(ctx context.Context, fh fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// Returns the requested file as bytes.
	filepath := n.currentPath()

	entry, found := fileIndex[filepath]
	if found == false {
		return nil, syscall.ENOENT
	}

	end := int(off) + len(dest)
	if end > len(entry.Data) {
		end = len(entry.Data)
	}
	return fuse.ReadResultData(entry.Data[off:end]), fs.OK
}

func (n *RssfsNode) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	// As rssfs is read-only, we can nop here.
	return fs.OK
}

func (n *RssfsNode) Open(ctx context.Context, mode uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	// Parses the file into a file struct.
	filepath := n.currentPath()

	/*
		if mode & syscall.S_IFREG == 0 {
			emoji.Printf(":bangbang: File mode not valid: (%d) != (%d)\n", mode, syscall.S_IFREG)
			return nil, 0, syscall.ENOENT
		}
	*/

	entry, found := fileIndex[filepath]
	if found == false {
		return nil, 0, syscall.ENOENT
	}

	fh = &RssfsNode{
		data: entry.Data,
	}

	return fh, 0, 0
}

func Mount(cfg RssfsConfig) {
	// Store the configuration globally.
	config = cfg

	// Mounts the feeds into our mountpoint.
	virtualRootPath := "/"

	rn := NewRssfsNode(virtualRootPath)

	sec := time.Second
	opts := &fs.Options{
		AttrTimeout:  &sec,
		EntryTimeout: &sec,
		MountOptions: fuse.MountOptions{
			AllowOther: true,
			Debug:      false,
			FsName:     "RSS File System",
		},
	}

	fs := fs.NewNodeFS(rn, opts)

	mountPoint := cfg.MountPoint

	server, err := fuse.NewServer(fs, mountPoint, &opts.MountOptions)
	if err != nil {
		// Whoops. Quit with a reasonable error message.
		emoji.Printf(":bangbang: rssfs could not be mounted: %s\n", err)
		user, userr := user.Current()
		if userr != nil {
			emoji.Println(":bangbang: Additionally, the current user could not be determined.")
			emoji.Printf(":bangbang: Is the mountpoint '%s' writable?\n", mountPoint)
		} else {
			emoji.Printf(":bangbang: Is the mountpoint '%s' writable by %s?\n", mountPoint, user.Username)
		}
		emoji.Println(":wave: rssfs will quit now.")
		os.Exit(1)
	}

	// Setting up the signals.
	quitChan := make(chan os.Signal)
	shutdownChan := make(chan struct{})

	signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(shutdownChan chan struct{}) {
		server.Serve()
		wg.Done()
	}(shutdownChan)

	emoji.Println(":rocket: Ready! Unmount to terminate.")

	<-quitChan
	close(shutdownChan)
	wg.Wait()
}
