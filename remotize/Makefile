include $(GOROOT)/src/Make.inc

TARG=remotize
GOFILES=remotize.go

REMOTIZED=$(ls *Remotized.go)

all: $(REMOTIZED)

install: $(REMOTIZED)

$(REMOTIZED): 
	@echo Prebuild
	@echo "package remotized" > dummyRemotized.go

GOFILES+=$(ls *Remotized.go)

include $(GOROOT)/src/Make.pkg
