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

#$(DIR)/remotize/misc/_obj $(DIR)/remotize/pipe/_obj

#$(DIR)/remotize/pipe/_obj:
#	cd $(DIR)/remotize/pipe/ && gomake install

#$(DIR)/remotize/misc/_obj:
#	cd $(DIR)/remotize/misc/ && gomake install

#$(DIR)/goremote/_obj: $(DIR)/remotize/tool/_obj
#	cd $(DIR)/goremote/ && gomake

