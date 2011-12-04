package dep

import (
	"gob"
	"os"
	"strings"
)

func init() {
	// Allow remotization (gob-ability) of my own error...
	gob.Register(new(myError))
}

// my error type
type myError struct {
	Err string
}

// Meets the Error interface
func (e *myError) String() string {
	return e.Err
}

// wrapError will wrap an os.Error to be myError and thus, be gob-able = sendable
func wrapError(e os.Error) os.Error {
	if e==nil {
		return nil
	}
	return &myError{e.String()}
}

// Stateless version of a (local) File Service
type FileService struct {

}

// Create returns a new File created by a given name
func (fs *FileService) Create(filename string) os.Error {
	_, e := os.Create(filename)
	return wrapError(e)
}

// Mkdir creates a new directory (and all subdirectories in between)
func (fs *FileService) Mkdir(filename string) os.Error {
	return wrapError(os.MkdirAll(filename, 0755))
}

// Remove will delete a file by a given name
func (fs *FileService) Remove(filename string) os.Error {
	return wrapError(os.Remove(filename))
}

// FileInfo returns the fileinfo for a give file name
func (fs *FileService) FileInfo(filename string) (*os.FileInfo, os.Error) {
	fi, e := os.Lstat(filename)
	return fi, wrapError(e)
}

// Rename will rename a directory or file
func (fs *FileService) Rename(oldname, newname string) os.Error {
	return wrapError(os.Rename(oldname, newname))
}

// ReadAt will read a filename at a given offset into a given byte array
func (fs *FileService) ReadAt(filename string, b []byte, off int64) (int, os.Error) {
	f, e := os.Open(filename)
	if e != nil {
		return 0, e
	}
	n, e := f.ReadAt(b, off)
	return n, wrapError(e)
}

// WriteAt will write the given bytes at a certain offset on filename
func (fs *FileService) WriteAt(filename string, b []byte, off int64) (int, os.Error) {
	f, e := os.Open(filename)
	if e != nil {
		return 0, e
	}
	defer gosync(f)
	n, e := f.WriteAt(b, off)
	return n, wrapError(e)
}

// ReadDir will list the directory contents
func (fs *FileService) Readdir(filename string, n int) ([]os.FileInfo, os.Error) {
	f, e := os.Open(filename)
	if e != nil {
		return nil, e
	}
	fis, e := f.Readdir(n)
	return fis, wrapError(e)
}

func gosync(f *os.File) {
	go func(*os.File) {
		f.Sync()
	}(f)
}

// Stateless version of a Process Servicer interface (local or remote)
type ProcessServicer interface {
	NewProcess(cmd string) (int, os.Error)
	Kill(id int) os.Error
	Signal(id int, sig os.Signal) os.Error
	Wait(id int, options int) (*os.Waitmsg, os.Error)
}

// Local Process Service
type ProcessService struct {

}

// NewProcess generates a new process running a certain command and returns its id or an Error
func (ps *ProcessService) NewProcess(cmd string) (int, os.Error) {
	argv := strings.Split(cmd, " ")
	name := argv[0]
	p, e := os.StartProcess(name, argv, &os.ProcAttr{"", nil, nil, nil})
	if e != nil {
		return -1, e
	}
	return p.Pid, nil
}

// Kill will end the process identified by id
func (ps *ProcessService) Kill(pid int) os.Error {
	p, e := os.FindProcess(pid)
	if e != nil {
		return e
	}
	return p.Kill()
}

// Signal will send a given signal to a process identified by id
func (ps *ProcessService) Signal(pid int, sig os.Signal) os.Error {
	p, e := os.FindProcess(pid)
	if e != nil {
		return e
	}
	return p.Signal(sig)
}

// Wait will wait a Process
func (ps *ProcessService) Wait(pid int, options int) (*os.Waitmsg, os.Error) {
	p, e := os.FindProcess(pid)
	if e != nil {
		return nil, e
	}
	return p.Wait(options)
}

