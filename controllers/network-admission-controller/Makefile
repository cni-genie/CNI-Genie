build:
	docker build --no-cache -t genie-network-admission-controller .
deploy-only:
	kubectl config use-context local || true
	kubectl delete -f deployment/service.yaml || true
	kubectl delete -f deployment/pod.yaml || true
	kubectl delete externaladmissionhookconfiguration genie-network-admission-controller || true
	kubectl delete -f deployment/service.yaml || true
	kubectl create -f deployment/service.yaml
	# It's necessary because the genie-network-admission-controller needs to access the extension-apiserver-authentication configmap
	kubectl delete clusterrolebinding genie-network-admission-controller || true
	kubectl delete -f deployment/pod.yaml || true
	kubectl create clusterrolebinding genie-network-admission-controller --clusterrole=cluster-admin --serviceaccount=default:default
	sleep 5
	kubectl create -f deployment/pod.yaml	
deploy: build deploy-only