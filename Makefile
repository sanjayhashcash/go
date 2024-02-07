# Docker build targets use an optional "TAG" environment
# variable can be set to use custom tag name. For example:
#   TAG=my-registry.example.com/keystore:dev make keystore
XDRS = xdr/Hcnet-SCP.x \
xdr/Hcnet-ledger-entries.x \
xdr/Hcnet-ledger.x \
xdr/Hcnet-overlay.x \
xdr/Hcnet-transaction.x \
xdr/Hcnet-types.x \
xdr/Hcnet-contract-env-meta.x \
xdr/Hcnet-contract-meta.x \
xdr/Hcnet-contract-spec.x \
xdr/Hcnet-contract.x \
xdr/Hcnet-internal.x \
xdr/Hcnet-contract-config-setting.x

XDRGEN_COMMIT=e2cac557162d99b12ae73b846cf3d5bfe16636de
XDR_COMMIT=bb54e505f814386a3f45172e0b7e95b7badbe969

.PHONY: xdr xdr-clean xdr-update

keystore:
	$(MAKE) -C services/keystore/ docker-build

ticker:
	$(MAKE) -C services/ticker/ docker-build

friendbot:
	$(MAKE) -C services/friendbot/ docker-build

aurora:
	$(MAKE) -C services/aurora/ binary-build

webauth:
	$(MAKE) -C exp/services/webauth/ docker-build

recoverysigner:
	$(MAKE) -C exp/services/recoverysigner/ docker-build

regulated-assets-approval-server:
	$(MAKE) -C services/regulated-assets-approval-server/ docker-build

gxdr/xdr_generated.go: $(XDRS)
	go run github.com/xdrpp/goxdr/cmd/goxdr -p gxdr -enum-comments -o $@ $(XDRS)
	gofmt -s -w $@

xdr/%.x:
	printf "%s" ${XDR_COMMIT} > xdr/xdr_commit_generated.txt
	curl -Lsf -o $@ https://raw.githubusercontent.com/hcnet/hcnet-xdr/$(XDR_COMMIT)/$(@F)

xdr/xdr_generated.go: $(XDRS)
	docker run -it --rm -v $$PWD:/wd -w /wd ruby /bin/bash -c '\
		gem install specific_install -v 0.3.8 && \
		gem specific_install https://github.com/hcnet/xdrgen.git -b $(XDRGEN_COMMIT) && \
		xdrgen \
			--language go \
			--namespace xdr \
			--output xdr/ \
			$(XDRS)'
	# No, you're not reading the following wrong. Apperantly, running gofmt twice required to complete it's formatting.
	gofmt -s -w $@
	gofmt -s -w $@

xdr: gxdr/xdr_generated.go xdr/xdr_generated.go

xdr-clean:
	rm xdr/*.x || true

xdr-update: xdr-clean xdr
