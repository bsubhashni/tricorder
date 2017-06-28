GOPATH  	= $(CURDIR)
AGENT   	= $(GOPATH)/cmd/agent
COORDINATOR = $(GOPATH)/cmd/coordinator

all:
	@cd $(AGENT) && go build -ldflags -s
	@cd $(COORDINATOR) && go build -ldflags -s
	@rm -rf bin && mkdir bin
	@cp $(AGENT)/agent bin/agent
	@cp $(COORDINATOR)/coordinator bin/coordinator
