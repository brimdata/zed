.PHONY: helm-install-fluentbit
helm-install-fluentbit:
	-kubectl create namespace logging
	@helm upgrade eks-fluentbit ../charts/eks-fluentbit \
		--install \
		--namespace logging \
		--set aws.region=us-east-2 \
		--set aws.clusterName=zq-test

.PHONY: helm-uninstall-fluentbit
helm-uninstall-fluentbit:
	@helm uninstall eks-fluentbit \
		--namespace logging

