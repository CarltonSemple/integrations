.PHONY: run clean

SUDO=$(shell docker info >/dev/null 2>&1 || echo "sudo -E")
EXE=a8plugin
IMAGE=a8-plugin

$(EXE): main.go
	#rm $(EXE)
	go build -v -o $(EXE)
	$(SUDO) docker build -t $(IMAGE) .

run: 
	docker build -t $(IMAGE) .
	$(SUDO) docker run --rm -it \
		--net=host \
		--name $(IMAGE) $(IMAGE)
		#-v /var/run/scope/plugins:/var/run/scope/plugins \
		#-v /var/run:/var/run \

clean:
	- rm $(EXE)
	- $(SUDO) docker rmi $(IMAGE)
