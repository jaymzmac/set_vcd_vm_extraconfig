.PHONY: build
build:
	for arch in amd64 arm64; do \
		for os in darwin linux windows; do \
			GOOS=$$os GOARCH=$$arch go build -o set_vcd_vm_extraconfig_$$os-$$arch set_vcd_vm_extraconfig.go ; \
		done \
	done