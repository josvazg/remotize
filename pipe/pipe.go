// Copyright 2010 Jose Luis Vázquez González josvazg@gmail.com
// Use of this source code is governed by a BSD-style

// This package is a covenience IO wrapper in case you need to connect two local
// processes so that you invoke remotized types also on a parent/child IPC pipe,
// and not only through the network. 
//
package pipe

import (
	"io"
	"os"
)

// Pipe for local invocations, parent/child process communications.
type Pipe struct {
	in  io.ReadCloser
	out io.WriteCloser
}

// Read from the pipe.
func (p *Pipe) Read(b []byte) (n int, err os.Error) {
	return p.in.Read(b)
}

// Write to the pipe.
func (p *Pipe) Write(b []byte) (n int, err os.Error) {
	return p.out.Write(b)
}

// Close will close both directions of the piped io.
func (p *Pipe) Close() os.Error {
	err := p.in.Close()
	if err != nil {
		return err
	}
	return p.out.Close()
}

// Prepare a ReadWriteCloser Pipe from a reader and a writer.
//
// This can be passed to rpc.NewClient to use RPCs over local pipe streams.
func PipeIO(in io.ReadCloser, out io.WriteCloser) *Pipe {
	return &Pipe{in, out}
}
