// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nullfs

import (
	"flag"
	"io"
	"net"
	"os"
	"testing"

	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/clnt"
)

var debug = flag.Int("debug", 0, "Debug level")

// It's recommended not to have a helper. But this is so much boiler plate.
func setup(msize uint32, failf func(...interface{})) (*clnt.Clnt, *clnt.Fid) {
	f := new(NullFS)
	f.Dotu = false
	f.Id = "ufs"
	f.Debuglevel = *debug
	if !f.Start(f) {
		failf("Can't happen: Starting the server failed")
	}

	l, err := net.Listen("unix", "")
	if err != nil {
		failf("net.Listen: want nil, got %v", err)
	}

	go func() {
		if err = f.StartListener(l); err != nil {
			failf("Can not start listener: %v", err)
		}
	}()

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt, err := clnt.Mount("unix", l.Addr().String(), "/", msize, user)

	if err != nil {
		failf("Attach: %v", err)
	}

	rootfid := clnt.Root
	return clnt, rootfid
}

func TestAttach(t *testing.T) {
	setup(9000, t.Fatal)
}
func TestAttachOpenReaddir(t *testing.T) {
	var err error
	clnt, rootfid := setup(9000, t.Fatal)

	dirfid := clnt.FidAlloc()
	if _, err = clnt.Walk(rootfid, dirfid, []string{}); err != nil {
		t.Fatalf("%v", err)
	}

	if err = clnt.Open(dirfid, 0); err != nil {
		t.Fatalf("%v", err)
	}
	var b []byte
	if b, err = clnt.Read(dirfid, 0, 64*1024); err != nil {
		t.Fatalf("%v", err)
	}
	var i, amt int
	var offset uint64
	err = nil
	for err == nil {
		if b, err = clnt.Read(dirfid, offset, 64*1024); err != nil {
			t.Fatalf("%v", err)
		}

		if len(b) == 0 {
			break
		}
		for b != nil && len(b) > 0 {
			if _, b, amt, err = ninep.UnpackDir(b, true); err != nil {
				t.Errorf("UnpackDir returns %v", err)
				break
			} else {
				i++
				offset += uint64(amt)
			}
		}
	}
	if i != len(dirQids) {
		t.Fatalf("Reading: got %d entries, wanted %d, err %v", i, len(dirQids), err)
	}

	t.Logf("-----------------------------> Alternate form, using readdir and File")
	// Alternate form, using readdir and File
	dirfile, err := clnt.FOpen(".", ninep.OREAD)
	if err != nil {
		t.Fatalf("%v", err)
	}
	i, amt, offset = 0, 0, 0
	err = nil

	for err == nil {
		d, err := dirfile.Readdir(64)
		if err != nil && err != io.EOF {
			t.Errorf("%v", err)
		}

		if len(d) == 0 {
			break
		}
		i += len(d)
		if i >= len(dirQids) {
			break
		}
	}
	if i != len(dirQids)-1 {
		t.Fatalf("Readdir: got %d entries, wanted %d", i, len(dirQids)-1)
	}
}

func TestNull(t *testing.T) {
	var err error
	clnt, rootfid := setup(9000, t.Fatal)

	d := clnt.FidAlloc()
	if _, err = clnt.Walk(rootfid, d, []string{"null"}); err != nil {
		t.Fatalf("Walk %v", err)
	}

	if err = clnt.Open(d, 0); err != nil {
		t.Fatalf("Open %v", err)
	}

	var b []byte
	if b, err = clnt.Read(d, 0, 64*1024); err != nil {
		t.Fatalf("Read %v", err)
	}
	if len(b) > 0 {
		t.Fatalf("Read of null: want 0, got %d bytes", len(b))
	}

	st, err := clnt.Stat(d)
	if err != nil {
		t.Fatalf("Stat %v", err)
	}

	if st.Name != "null" {
		t.Fatalf("Stat: want 'null', got %v", st.Name)
	}
	if st.Mode != 0666 {
		t.Fatalf("Stat: want 0777, got %o", st.Mode)
	}

}

func TestZero(t *testing.T) {
	var err error
	clnt, rootfid := setup(9000, t.Fatal)

	d := clnt.FidAlloc()
	if _, err = clnt.Walk(rootfid, d, []string{"zero"}); err != nil {
		t.Fatalf("%v", err)
	}

	if err = clnt.Open(d, 0); err != nil {
		t.Fatalf("%v", err)
	}

	var b []byte
	if b, err = clnt.Read(d, 0, 64*1024); err != nil {
		t.Fatalf("%v", err)
	}
	if len(b) == 0 {
		t.Fatalf("Read of null: want > 0, got %d bytes", len(b))
	}

}

func BenchmarkNull(b *testing.B) {
	clnt, rootfid := setup(9000, b.Fatal)
	d := clnt.FidAlloc()
	if _, err := clnt.Walk(rootfid, d, []string{"null"}); err != nil {
		b.Fatalf("%v", err)
	}

	if err := clnt.Open(d, 0); err != nil {
		b.Fatalf("%v", err)
	}

	for i := 0; i < b.N; i++ {
		if _, err := clnt.Read(d, 0, 64*1024); err != nil {
			b.Fatalf("%v", err)
		}
	}

}

func benchZero(b *testing.B, msize, iosize uint32) {
	clnt, rootfid := setup(msize, b.Fatal)
	d := clnt.FidAlloc()
	if _, err := clnt.Walk(rootfid, d, []string{"zero"}); err != nil {
		b.Fatalf("%v", err)
	}

	if err := clnt.Open(d, 0); err != nil {
		b.Fatalf("%v", err)
	}

	for i := 0; i < b.N; i++ {
		for tot := uint32(0); tot < iosize; {
			if n, err := clnt.Read(d, 0, iosize); err != nil {
				b.Fatalf("%v: only got %d of %d bytes", err, len(n), iosize)
			} else {
				tot += uint32(len(n))
			}
		}
	}
}

func BenchmarkZero8KPkt9KMsize(b *testing.B) {
	benchZero(b, 9000, 8192)
}

func BenchmarkZero1MPkt1MMsize(b *testing.B) {
	benchZero(b, 1<<20, 1<<20)
}

func BenchmarkZero1Mpkt9KMsize(b *testing.B) {
	benchZero(b, 9000, 1<<20)
}

func BenchmarkZero8KPkt1MMsize(b *testing.B) {
	benchZero(b, 1<<20, 8192)

}

/*
func BenchmarkRootWalk(b *testing.B) {
	nullfs := new(nullfs.Nullfs)
	nullfs.Dotu = false
	nullfs.Id = "nullfs"
	nullfs.Debuglevel = *debug
	nullfs.Msize = 8192
	nullfs.Start(nullfs)

	l, err := net.Listen("unix", "")
	if err != nil {
		b.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	go func() {
		if err = nullfs.StartListener(l); err != nil {
			b.Fatalf("Can not start listener: %v", err)
		}
		b.Fatalf("Listener returned")
	}()
	var conn net.Conn
	if conn, err = net.Dial("unix", srvAddr); err != nil {
		b.Fatalf("%v", err)
	}

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt := NewClnt(conn, 8192, false)
	rootfid, err := clnt.Attach(nil, user, "/")
	if err != nil {
		b.Fatalf("%v", err)
	}

	for i := 0; i < b.N; i++ {
		f := clnt.FidAlloc()
		if _, err = clnt.Walk(rootfid, f, []string{"bin"}); err != nil {
			b.Fatalf("%v", err)
		}
	}
}
func BenchmarkRootWalkBadFid(b *testing.B) {
	nullfs := new(nullfs.Nullfs)
	nullfs.Dotu = false
	nullfs.Id = "nullfs"
	nullfs.Debuglevel = *debug
	nullfs.Msize = 8192
	nullfs.Start(nullfs)

	l, err := net.Listen("unix", "")
	if err != nil {
		b.Fatalf("Can not start listener: %v", err)
	}
	srvAddr := l.Addr().String()
	go func() {
		if err = nullfs.StartListener(l); err != nil {
			b.Fatalf("Can not start listener: %v", err)
		}
		b.Fatalf("Listener returned")
	}()
	var conn net.Conn
	if conn, err = net.Dial("unix", srvAddr); err != nil {
		b.Fatalf("%v", err)
	}

	user := ninep.OsUsers.Uid2User(os.Geteuid())
	clnt := NewClnt(conn, 8192, false)
	rootfid, err := clnt.Attach(nil, user, "/")
	if err != nil {
		b.Fatalf("%v", err)
	}

	rootfid.Fid++
	for i := 0; i < b.N; i++ {
		if _, err = clnt.Walk(rootfid, rootfid, []string{"bin"}); err == nil {
			b.Fatalf("Did not get an expected error on walking a bad fid!")
		}
	}
}
*/
