// Copyright 2009 The Ninep Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os/user"

	"github.com/lionkov/ninep"
	"github.com/lionkov/ninep/srv/ufs"
)

var (
	debug = flag.Int("d", 0, "print debug messages")
	addr = flag.String("addr", ":5640", "network address")
)

func main() {
	flag.Parse()
	ufs := ufs.New()
	ufs.Dotu = true
	ufs.Id = "ufs"
	ufs.Debuglevel = *debug
	ufs.Start(ufs)

	u, uerr := user.Current()
	if uerr != nil {
		log.Fatalln(uerr)
	}
	u.Username = "glenda"
	u.HomeDir = "/usr/glenda"
	ninep.OsUsers.Simulate(u)

	err := ufs.StartNetListener("tcp", *addr)
	if err != nil {
		log.Println(err)
	}
}
