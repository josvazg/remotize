include $(GOROOT)/src/Make.inc

TARG=github.com/josvazg/remotize
GOFILES=remotize.go

include $(GOROOT)/src/Make.pkg

clean: cleandeps

cleandeps:
	gomake -C pipe clean
	gomake -C misc clean
	gomake -C tool clean
	gomake -C goremote clean

all: buildeps

buildeps:
	gomake -C pipe install
	gomake -C misc install
	gomake test install
	gomake -C tool test install
	rm tool/remotized*.go
#	cd goremote && gotest
	gomake -C goremote install

