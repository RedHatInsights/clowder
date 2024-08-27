def secrets = [
    [path: params.VAULT_PATH_QUAY_PUSH, engineVersion: 1, secretValues: [
        [envVar: 'QUAY_USER', vaultKey: 'user'],
        [envVar: 'QUAY_TOKEN', vaultKey: 'token']]],
    [path: params.VAULT_PATH_QUAY_TOKEN, engineVersion: 1, secretValues: [
        [envVar: 'QUAY_API_TOKEN', vaultKey: 'api-token']]],
    [path: params.VAULT_PATH_MINIKUBE, engineVersion: 1, secretValues: [
        [envVar: 'MINIKUBE_SSH_KEY', vaultKey: 'private-key'],
        [envVar: 'MINIKUBE_HOST', vaultKey: 'hostname'],
        [envVar: 'MINIKUBE_USER', vaultKey: 'user'],
        [envVar: 'MINIKUBE_ROOTDIR', vaultKey: 'rootdir']]]
]
def configuration = [vaultUrl: params.VAULT_ADDRESS, vaultCredentialId: params.VAULT_CREDS_ID, engineVersion: 1]

pipeline {
    agent { label 'rhel8' }
    options {
        timestamps()
    }

    environment {
        CLOWDER_VERSION=sh(script: "git describe --tags", returnStdout: true).trim()

        BASE_TAG=sh(script:"cat go.mod go.sum Dockerfile.base | sha256sum  | head -c 8", returnStdout: true)
        BASE_IMG="quay.io/cloudservices/clowder-base:${BASE_TAG}"

        IMAGE_TAG=sh(script:"git rev-parse --short=8 HEAD", returnStdout: true).trim()
        IMAGE_NAME="quay.io/cloudservices/clowder"

        CICD_URL="https://raw.githubusercontent.com/RedHatInsights/cicd-tools/alternative-cicd-tools"
        HELPER_FUNCTIONS="${CICD_URL}/helpers/general.sh"
        BOOTSTRAP_FUNCTIONS="${CICD_URL}/src/bootstrap.sh"
        CURR_TIME=sh(script: "date +%s", returnStdout: true).trim()
    }

    stages {
        stage('Did you run pre-push?') {
            environment {
                TEST_CONTAINER="clowder-ci-prepush-test-${IMAGE_TAG}-${CURR_TIME}"
            }
            steps {
                withVault([configuration: configuration, vaultSecrets: secrets]) {
                    sh '''
                        docker login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io
                        docker build -f Dockerfile.test --build-arg BASE_IMAGE=${BASE_IMG} -t $TEST_CONTAINER .
                        docker run -i $TEST_CONTAINER \
                            ./githooks/prepush.sh
                    '''
                    }
                }
            
            post {
                always {
                    sh 'docker rm -f $TEST_CONTAINER'
                    }
                }
            }

        stage('Build and Push Base Image') {
            steps {
                withVault([configuration: configuration, vaultSecrets: secrets]) {
                    sh './ci/build_push_base_img.sh'
                }
            }
        }

        stage('Initial Setup') {
            steps {
                sh '''
                    make envtest
                    make update-version
                '''
            }
        }

        stage('Run Tests') {
            parallel {
                stage('Unit Tests') {
                    environment {
                        TEST_CONTAINER="clowder-ci-unit-tests-${IMAGE_TAG}-${CURR_TIME}"
                    }
                    steps {
                        withVault([configuration: configuration, vaultSecrets: secrets]) {
                            sh './ci/unit_tests.sh'
                        }
                    }

                    post {
                        always {
                            sh 'docker rm -f $TEST_CONTAINER'
                        }
                    }
                }

                stage('Minikube E2E Tests') {
                    environment {
                        CONTAINER_NAME="clowder-ci-minikube-e2e-tests-${IMAGE_TAG}-${CURR_TIME}"
                    }
                    steps {
                        withVault([configuration: configuration, vaultSecrets: secrets]) {
                            sh './ci/minikube_e2e_tests.sh'
                        }
                    }

                    post {
                        always {
                            sh 'docker rm -f $CONTAINER_NAME'
                            archiveArtifacts artifacts: 'artifacts/**/*', fingerprint: true
                            junit skipPublishingChecks: true, testResults: 'artifacts/junit-*.xml'
                        }
                    }
                }
            }
        }  
    }
}
