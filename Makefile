# I hope this can go away when go manages to compile without makefiles
DIR=src/josvazg

GOPATH=$(HOME)/remotize

all: $(DIR)/remotize/tool/_test
	rm -f $(DIR)/remotize/tool/remotize*.go
	goinstall josvazg/remotize/pipe
	goinstall josvazg/goremote

clean:
	cd $(DIR)/goremote && gomake clean
	cd $(DIR)/remotize && gomake clean
	cd $(DIR)/remotize/pipe && gomake clean
	cd $(DIR)/remotize/misc && gomake clean
	cd $(DIR)/remotize/tool && gomake clean

$(DIR)/remotize/tool/_test: $(DIR)/remotize/_test
	cd $(DIR)/remotize/tool/ && gotest

$(DIR)/remotize/_test: 
	cd $(DIR)/remotize/ && gotest

