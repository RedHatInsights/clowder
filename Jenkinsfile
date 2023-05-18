pipeline {
    agent { label 'insights' }
    options {
        timestamps()
    }
    stages {
        stage('no-op') {
            steps {
                sh "echo 'hello world!'"
            }
        }
    }
}

