include $(GOROOT)/src/Make.inc

TARG=github.com/josvazg/remotize
GOFILES=remotize.go

include $(GOROOT)/src/Make.pkg

clean: cleandeps

cleandeps:
	gomake -C tool clean
	gomake -C goremote clean
	gomake -C sample/dep clean
	gomake -C sample clean

all: $(GOROOT)/src/Make.rpkg test sample/_obj

$(GOROOT)/src/Make.rpkg: tool/Make.rpkg
	cp tool/Make.rpkg $(GOROOT)/src/Make.rpkg

tool/_obj: _obj tool/gen.go tool/detect.go tool/tool_test.go
	gomake -C tool test install
	rm tool/remotized*.go

goremote/_obj:  goremote/goremote.go tool/_obj
	#cd goremote && gotest
	gomake -C goremote install

sample/dep/_obj: sample/dep/dep.go
	gomake -C sample/dep install

sample/_obj: sample/dep/_obj goremote/_obj sample/sample.go sample/sample_test.go
	gomake -C sample test


