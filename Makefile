include $(GOROOT)/src/Make.inc

TARG=github.com/josvazg/remotize
GOFILES=remotize.go

include $(GOROOT)/src/Make.pkg

clean: cleandeps

cleandeps:
	gomake -C pipe clean
	gomake -C tool clean
	gomake -C goremote clean

all: test buildeps

buildeps:  tool/_obj pipe/_obj
#	cd goremote && gotest
	gomake -C goremote install

tool/_obj: _obj tool/tool.go
	gomake -C tool test install
	rm tool/remotized*.go

pipe/_obj: pipe/pipe.go
	gomake -C pipe install
