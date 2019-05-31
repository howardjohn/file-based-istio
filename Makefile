format:
	dirs=$(shell go list  -f "{{.Dir}}" ./...)
	for d in ${dirs}; do
		echo $d
	done
	# goimports -w github.com/howardjohn/file-based-istio