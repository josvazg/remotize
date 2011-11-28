package dep

import (
	"os"
	"strings"
)

// Stateless version of a (local) File Service
type FileService struct {

}

// Create returns a new File created by a given name
func (fs *FileService) Create(filename string) os.Error {
	_, e := os.Create(filename)
	return e
}

// Mkdir creates a new directory (and all subdirectories in between)
func (fs *FileService) Mkdir(filename string) os.Error {
	return os.MkdirAll(filename, 0)
}

// Remove will delete a file by a given name
func (fs *FileService) Remove(filename string) os.Error {
	return os.Remove(filename)
}

// FileInfo returns the fileinfo for a give file name
func (fs *FileService) FileInfo(filename string) (fi *os.FileInfo, err os.Error) {
	return os.Lstat(filename)
}

// Rename will rename a directory or file
func (fs *FileService) Rename(oldname, newname string) os.Error {
	return os.Rename(oldname, newname)
}

// ReadAt will read a filename at a given offset into a given byte array
func (fs *FileService) ReadAt(filename string, b []byte, off int64) (int, os.Error) {
	f, e := os.Open(filename)
	if e != nil {
		return 0, e
	}
	return f.ReadAt(b, off)
}

// WriteAt will write the given bytes at a certain offset on filename
func (fs *FileService) WriteAt(filename string, b []byte, off int64) (int, os.Error) {
	f, e := os.Open(filename)
	if e != nil {
		return 0, e
	}
	defer gosync(f)
	return f.WriteAt(b, off)
}

// ReadDir will list the directory contents
func (fs *FileService) ReadDir(filename string, n int) ([]os.FileInfo, os.Error) {
	f, e := os.Open(filename)
	if e != nil {
		return nil, e
	}
	return f.Readdir(n)
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
func NewProcess(cmd string) (int, os.Error) {
	argv := strings.Split(cmd, " ")
	name := argv[0]
	p, e := os.StartProcess(name, argv, &os.ProcAttr{"", nil, nil, nil})
	if e != nil {
		return -1, e
	}
	return p.Pid, nil
}

// Kill will end the process identified by id
func Kill(pid int) os.Error {
	p, e := os.FindProcess(pid)
	if e != nil {
		return e
	}
	return p.Kill()
}

// Signal will send a given signal to a process identified by id
func Signal(pid int, sig os.Signal) os.Error {
	p, e := os.FindProcess(pid)
	if e != nil {
		return e
	}
	return p.Signal(sig)
}

// Wait will wait a Process
func Wait(pid int, options int) (*os.Waitmsg, os.Error) {
	p, e := os.FindProcess(pid)
	if e != nil {
		return nil, e
	}
	return p.Wait(options)
}
