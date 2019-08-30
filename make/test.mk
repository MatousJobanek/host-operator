############################################################
#
# (local) Tests
#
############################################################

.PHONY: test
## runs the tests without coverage and excluding E2E tests
test:
	@-echo "printing out"
	@-git config --get remote.origin.url
	@-git branch
	@echo "running the tests without coverage and excluding E2E tests..."
	$(Q)go test ${V_FLAG} -race $(shell go list ./... | grep -v /test/e2e) -failfast

############################################################
#
# OpenShift CI Tests with Coverage
#
############################################################

# Output directory for coverage information
COV_DIR = $(OUT_DIR)/coverage

.PHONY: test-with-coverage
## runs the tests with coverage
test-with-coverage:
	@echo "running the tests with coverage..."
	@-mkdir -p $(COV_DIR)
	@-rm $(COV_DIR)/coverage.txt
	$(Q)go test -vet off ${V_FLAG} $(shell go list ./... | grep -v /test/e2e) -coverprofile=$(COV_DIR)/coverage.txt -covermode=atomic ./...

.PHONY: upload-codecov-report
# Uploads the test coverage reports to codecov.io. 
# DO NOT USE LOCALLY: must only be called by OpenShift CI when processing new PR and when a PR is merged! 
upload-codecov-report: 
	# Upload coverage to codecov.io. Since we don't run on a supported CI platform (Jenkins, Travis-ci, etc.), 
	# we need to provide the PR metadata explicitely using env vars used coming from https://github.com/openshift/test-infra/blob/master/prow/jobs.md#job-environment-variables
	# 
	# Also: not using the `-F unittests` flag for now as it's temporarily disabled in the codecov UI 
	# (see https://docs.codecov.io/docs/flags#section-flags-in-the-codecov-ui)
	env
ifneq ($(PR_COMMIT), null)
	@echo "uploading test coverage report for pull-request #$(PULL_NUMBER)..."
	bash <(curl -s https://codecov.io/bash) \
		-t $(CODECOV_TOKEN) \
		-f $(COV_DIR)/coverage.txt \
		-C $(PR_COMMIT) \
		-r $(REPO_OWNER)/$(REPO_NAME) \
		-P $(PULL_NUMBER) \
		-Z
else
	@echo "uploading test coverage report after PR was merged..."
	bash <(curl -s https://codecov.io/bash) \
		-t $(CODECOV_TOKEN) \
		-f $(COV_DIR)/coverage.txt \
		-C $(BASE_COMMIT) \
		-r $(REPO_OWNER)/$(REPO_NAME) \
		-Z
endif

CODECOV_TOKEN := "e0747034-8ed2-4165-8d0b-3015d94307f9"
REPO_OWNER := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].org')
REPO_NAME := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].repo')
BASE_COMMIT := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].base_sha')
PR_COMMIT := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].pulls[0].sha')
PULL_NUMBER := $(shell echo $$CLONEREFS_OPTIONS | jq '.refs[0].pulls[0].number')

MEMBER_NS := member-operator-$(shell date +'%s')
HOST_NS := host-operator-$(shell date +'%s')

###########################################################
#
# End-to-end Tests
#
###########################################################

AUTHOR_LINK := $(shell jq -r '.refs[0].pulls[0].author_link' <<< $${CLONEREFS_OPTIONS} | tr -d '[:space:]')
PULL_SHA := $(shell jq -r '.refs[0].pulls[0].sha' <<< $${CLONEREFS_OPTIONS} | tr -d '[:space:]')

.PHONY: test-e2e
test-e2e:
ifeq ($(E2E_REPO_PATH),)
	$(eval E2E_REPO_PATH = /tmp/toolchain-e2e)
	rm -rf ${E2E_REPO_PATH}
	# cloning here as don't want to maintain it for every single change in deploy directory of member-operator
	git clone https://github.com/codeready-toolchain/toolchain-e2e.git --depth 1 ${E2E_REPO_PATH}
endif
	@-echo "printing out"
	curl ${AUTHOR_LINK}/host-operator.git/info/refs?service=git-upload-pack --output - /dev/null 2>&1 | grep -a ${PULL_SHA} | awk '{print $$2}'
	$(eval BRANCH_REF := $(shell curl ${AUTHOR_LINK}/host-operator.git/info/refs?service=git-upload-pack --output - /dev/null 2>&1 | grep -a ${PULL_SHA} | awk '{print $$2}'))
	echo ${BRANCH_REF}
	$(eval EXISTS := $(shell curl ${AUTHOR_LINK}/toolchain-e2e.git/info/refs?service=git-upload-pack --output - /dev/null 2>&1 | grep -a ${BRANCH_REF} | awk '{print $$2}'))
	$(eval BRANCH_NAME := $(shell echo ${BRANCH_REF} | awk -F'/' '{print $$3}'))
	echo ${BRANCH_REF} | awk -F'/' '{print $3}'
	echo name ${BRANCH_NAME}
	git --git-dir=${E2E_REPO_PATH}/.git --work-tree=${E2E_REPO_PATH} remote add external ${AUTHOR_LINK}/toolchain-e2e.git
	git --git-dir=${E2E_REPO_PATH}/.git --work-tree=${E2E_REPO_PATH} fetch external ${BRANCH_REF}
	git --git-dir=${E2E_REPO_PATH}/.git --work-tree=${E2E_REPO_PATH} merge FETCH_HEAD
	$(MAKE) -C ${E2E_REPO_PATH} test-e2e TMP_HOST_PATH=${PWD}



.PHONY: print-logs
print-logs:
	@echo "=====================================================================================" &
	@echo "================================ Host cluster logs =================================="
	@echo "====================================================================================="
	@oc logs deployment.apps/host-operator --namespace $(HOST_NS)
	@echo "====================================================================================="
	@echo "================================ Member cluster logs ================================"
	@echo "====================================================================================="
	@oc logs deployment.apps/member-operator --namespace $(MEMBER_NS)
	@echo "====================================================================================="

.PHONY: e2e-setup
e2e-setup: is-minishift
	oc new-project $(HOST_NS) --display-name e2e-tests
	oc apply -f ./deploy/service_account.yaml
	oc apply -f ./deploy/role.yaml
	oc apply -f ./deploy/role_binding.yaml
	oc apply -f ./deploy/cluster_role.yaml
	sed -e 's|REPLACE_NAMESPACE|${HOST_NS}|g' ./deploy/cluster_role_binding.yaml  | oc apply -f -
	oc apply -f deploy/crds
	sed -e 's|REPLACE_IMAGE|${IMAGE_NAME}|g' ./deploy/operator.yaml  | oc apply -f -

.PHONY: is-minishift
is-minishift:
ifeq ($(OPENSHIFT_BUILD_NAMESPACE),)
	$(info logging as system:admin")
	$(shell echo "oc login -u system:admin")
	$(eval IMAGE_NAME := docker.io/${GO_PACKAGE_ORG_NAME}/${GO_PACKAGE_REPO_NAME}:${GIT_COMMIT_ID_SHORT})
	$(shell echo "make docker-image")
else
	$(eval IMAGE_NAME := registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:host-operator)
endif

.PHONY: setup-kubefed
setup-kubefed:
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/toolchain-common/master/scripts/add-cluster.sh | bash -s -- -t member -mn $(MEMBER_NS) -hn $(HOST_NS) -s
	curl -sSL https://raw.githubusercontent.com/codeready-toolchain/toolchain-common/master/scripts/add-cluster.sh | bash -s -- -t host -mn $(MEMBER_NS) -hn $(HOST_NS) -s

.PHONY: e2e-cleanup
e2e-cleanup:
	oc delete project ${MEMBER_NS} ${HOST_NS} --wait=false || true

.PHONY: clean-e2e-namespaces
clean-e2e-namespaces:
	$(Q)-oc get projects --output=name | grep -E "(member|host)\-operator\-[0-9]+" | xargs oc delete

###########################################################
#
# Deploying Member Operator in Openshift CI Environment
#
###########################################################

.PHONY: deploy-member
deploy-member:
	rm -rf /tmp/member-operator
	# cloning here as don't want to maintain it for every single change in deploy directory of member-operator
	git clone https://github.com/codeready-toolchain/member-operator.git --depth 1 /tmp/member-operator
	oc new-project $(MEMBER_NS)
	oc apply -f /tmp/member-operator/deploy/service_account.yaml
	oc apply -f /tmp/member-operator/deploy/role.yaml
	oc apply -f /tmp/member-operator/deploy/role_binding.yaml
	oc apply -f /tmp/member-operator/deploy/cluster_role.yaml
	cat /tmp/member-operator/deploy/cluster_role_binding.yaml | sed s/\REPLACE_NAMESPACE/$(MEMBER_NS)/ | oc apply -f -
	oc apply -f /tmp/member-operator/deploy/crds
	sed -e 's|REPLACE_IMAGE|registry.svc.ci.openshift.org/codeready-toolchain/member-operator-v0.1:member-operator|g' /tmp/member-operator/deploy/operator.yaml  | oc apply -f -
