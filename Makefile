docker:
	docker build . --platform=linux/amd64 -t docker.io/shaunagostinho/netwatcher-agent:latest
	docker push docker.io/shaunagostinho/netwatcher-agent:latest

helm-install:
	cd helm
	helm install netwatcher-agent --generate-name --set user_id=beavis --set user_pin=1337