package squashfs

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func cmpFileInfo(got os.FileInfo, want FileInfo) error {
	if got, want := got.Name(), want.name; got != want {
		return fmt.Errorf("unexpected file name: got %q, want %q", got, want)
	}
	if got, want := got.Size(), want.size; got != want {
		return fmt.Errorf("unexpected size: got %d, want %d", got, want)
	}
	if got, want := got.IsDir(), want.mode.IsDir(); got != want {
		return fmt.Errorf("IsDir: got %v, want %v", got, want)
	}
	if got, want := got.ModTime(), want.modTime; !got.Equal(want) {
		return fmt.Errorf("IsDir: got %v, want %v", got, want)
	}

	return nil
}

func TestReaddir(t *testing.T) {
	t.Parallel()
	// TODO: ship testdata files generated by mksquashfs
	f, err := os.Open("/home/michael/distri/build/distri/pkg/ack-amd64-2.24.squashfs")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rd, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	fis, err := rd.Readdir(rd.RootInode())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 2; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	if err := cmpFileInfo(fis[0], FileInfo{
		name:    "bin",
		size:    26,
		mode:    0555 | os.ModeDir,
		modTime: time.Unix(1555577972, 0), // stat -c %Y /ro/ack-amd64-2.24/bin
	}); err != nil {
		t.Fatal(err)
	}

	if err := cmpFileInfo(fis[1], FileInfo{
		name:    "out",
		size:    48,
		mode:    0555 | os.ModeDir,
		modTime: time.Unix(1555577972, 0), // stat -c %Y /ro/ack-amd64-2.24/out
	}); err != nil {
		t.Fatal(err)
	}

	fis, err = rd.Readdir(fis[0].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 1; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	if err := cmpFileInfo(fis[0], FileInfo{
		name:    "ack",
		size:    5661,
		mode:    0755,
		modTime: time.Unix(1555577972, 0), // stat -c %Y /ro/ack-amd64-2.24/bin/ack
	}); err != nil {
		t.Fatal(err)
	}
}

// TestReaddirSmoke is a smoke-test, reading the root directories of SquashFS
// images which are known to trigger code paths which were buggy.
func TestReaddirSmoke(t *testing.T) {
	t.Parallel()

	for _, fn := range []string{
		// bash exercises the code path where an inode is split across metadata
		// blocks.
		"/home/michael/distri/build/distri/pkg/bash-amd64-4.4.18.squashfs",

		// cmake exercises the code path where the root directory entries are
		// located outside of the first block.
		"/home/michael/distri/build/distri/pkg/cmake-amd64-3.12.2.squashfs",
	} {
		// TODO: ship testdata files generated by mksquashfs
		f, err := os.Open(fn)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		rd, err := NewReader(f)
		if err != nil {
			t.Fatal(err)
		}

		fis, err := rd.Readdir(rd.RootInode())
		if err != nil {
			t.Fatal(err)
		}

		if got, want := len(fis), 2; got != want {
			t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
		}
	}
}

func TestReaddirEmpty(t *testing.T) {
	t.Parallel()
	// TODO: ship testdata files generated by mksquashfs
	f, err := os.Open("/home/michael/distri/build/distri/pkg/zlib-amd64-1.2.11.squashfs")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rd, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	fis, err := rd.Readdir(rd.RootInode())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 2; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	if err := cmpFileInfo(fis[0], FileInfo{
		name:    "bin",
		size:    3,
		mode:    0555 | os.ModeDir,
		modTime: time.Unix(1555577479, 0), // stat -c %Y /ro/zlib-amd64-1.2.11/bin
	}); err != nil {
		t.Fatal(err)
	}

	fis, err = rd.Readdir(fis[0].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 0; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}
}

func TestReaddirSymlink(t *testing.T) {
	t.Parallel()
	// TODO: ship testdata files generated by mksquashfs
	f, err := os.Open("/home/michael/distri/build/distri/pkg/zlib-amd64-1.2.11.squashfs")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rd, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	fis, err := rd.Readdir(rd.RootInode())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 2; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	if err := cmpFileInfo(fis[1], FileInfo{
		name:    "out",
		size:    54,
		mode:    0555 | os.ModeDir,
		modTime: time.Unix(1555577479, 0), // stat -c %Y /ro/zlib-amd64-1.2.11/out
	}); err != nil {
		t.Fatal(err)
	}

	fis, err = rd.Readdir(fis[1].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 3; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	if err := cmpFileInfo(fis[1], FileInfo{
		name:    "lib",
		size:    100,
		mode:    0555 | os.ModeDir,
		modTime: time.Unix(1555577479, 0), // stat -c %Y /ro/zlib-amd64-1.2.11/out/lib
	}); err != nil {
		t.Fatal(err)
	}

	fis, err = rd.Readdir(fis[1].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 5; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	if err := cmpFileInfo(fis[1], FileInfo{
		name:    "libz.so",
		size:    14,
		mode:    0555 | os.ModeSymlink,
		modTime: time.Unix(1555577479, 0), // stat -c %Y /ro/zlib-amd64-1.2.11/out/lib/libz.so
	}); err != nil {
		t.Fatal(err)
	}

	// TODO: readlink
	target, err := rd.ReadLink(fis[1].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := target, "libz.so.1.2.11"; got != want {
		t.Fatalf("ReadLink(libz.so): got %q, want %q", got, want)
	}
}

func TestReadfile(t *testing.T) {
	t.Parallel()
	// TODO: ship testdata files generated by mksquashfs
	f, err := os.Open("/home/michael/distri/build/distri/pkg/ack-amd64-2.24.squashfs")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rd, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}

	fis, err := rd.Readdir(rd.RootInode())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 2; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	fis, err = rd.Readdir(fis[0].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(fis), 1; got != want {
		t.Fatalf("unexpected number of directory entries: got %d, want %d", got, want)
	}

	r, err := rd.FileReader(fis[0].Sys().(*FileInfo).Inode)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		h := md5.New()
		if _, err := io.Copy(h, r); err != nil {
			t.Fatal(err)
		}
		sum := fmt.Sprintf("%x", h.Sum(nil))
		if got, want := sum, "c98921729b3d7f36c8b46fa354c9131a"; got != want {
			t.Fatalf("md5(bin/ack): got %s, want %s", got, want)
		}
	}
}

// TODO: add test exercising ldirInodeHeader, e.g. '/mnt/loop/ca-certificates-3.39/buildoutput/etc/ssl'

func TestReadXattr(t *testing.T) {
	t.Parallel()

	// TODO: generate a smaller version of this file
	f, err := os.Open("testdata/xattr.squashfs")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rd, err := NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		Path string
		Want []Xattr
	}{
		{
			Path: "mtr-packet",
			Want: []Xattr{
				{
					Type:     XattrTypeSecurity,
					FullName: "security.capability",
					Value:    []byte{1, 0, 0, 2, 0, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
			},
		},
		{
			Path: "gnome-keyring-daemon",
			Want: []Xattr{
				{
					Type:     XattrTypeSecurity,
					FullName: "security.capability",
					Value:    []byte{1, 0, 0, 2, 0, 0x40, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
			},
		},
	} {
		inode, err := rd.LookupPath(tt.Path)
		if err != nil {
			t.Fatal(err)
		}
		xattrs, err := rd.ReadXattrs(inode)
		if err != nil {
			t.Fatalf("ReadXattrs(%v): %v", inode, err)
		}
		if diff := cmp.Diff(tt.Want, xattrs); diff != "" {
			t.Fatalf("unexpected ReadXattrs result: diff (-want +got):\n%s", diff)
		}
	}
}
