# I hope this can go away when go manages to compile without makefiles
DIR=src/josvazg

GOPATH=$(HOME)/remotize

all: $(DIR)/goremote/_obj
	@echo GOPATH=$(GOPATH)
	export GOPATH=$(GOPATH)
	goinstall $(DIR)/goremote

clean:
	cd $(DIR)/goremote && gomake clean
	cd $(DIR)/remotize && gomake clean
	cd $(DIR)/remotize/pipe && gomake clean
	cd $(DIR)/remotize/misc && gomake clean
	cd $(DIR)/remotize/tool && gomake clean
	

$(DIR)/remotize/pipe/_obj:
	cd $(DIR)/remotize/pipe/ && gomake install

$(DIR)/remotize/misc/_obj:
	cd $(DIR)/remotize/misc/ && gomake install

$(DIR)/remotize/_obj: $(DIR)/remotize/misc/_obj $(DIR)/remotize/pipe/_obj
	cd $(DIR)/remotize/ && gotest && gomake install

$(DIR)/remotize/tool/_obj: $(DIR)/remotize/_obj
	cd $(DIR)/remotize/tool/ && gotest && gomake install

$(DIR)/goremote/_obj: $(DIR)/remotize/tool/_obj
	cd $(DIR)/goremote/ && gomake

