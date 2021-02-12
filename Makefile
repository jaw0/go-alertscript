
ROOT != pwd
GO=env GOBIN=$(ROOT)/bin go

all:
	cd cmd/alertscript; $(GO) install
