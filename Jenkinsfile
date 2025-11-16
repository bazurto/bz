def imageName = 'bzbuilder'

def dockerfile = '''
FROM golang:1.25
RUN apt-get update && apt-get upgrade
RUN apt-get install -y build-essential git && mkdir /work
WORKDIR /work
CMD ["bash"]
'''
def tmpDockerfile = null

pipeline {
    agent any
    stages {
        stage('prepare') {
            steps {
                script {
                    def buildDir = ${env.WORKSPACE}/build-${env.BUILD_NUMBER}
                    tmpDockerfile = "${buildDir}/Dockerfile.tmp1"
                    writeFile file: tmpDockerfile, text: dockerfile
                }
                sh "docker build -t ${imageName} . -f ${tmpDockerfile}"
            }
        }
        stage('Build') {
            steps {
                sh "echo '---------------------------' && echo \$(whoami) && echo '---------------------------'"
                sh "echo '---------------------------' && echo \$(pwd) && echo '---------------------------'"
                sh 'docker run --rm -v $PWD:/work -w /work '+imageName+' make'
            }
        }
        stage('Test') {
            steps {
                sh 'docker run --rm -v $PWD:/work -w /work '+imageName+' make test'
            }
        }
        // stage('Deploy') {
        //     steps {
        //         echo 'Deploying....'
        //     }
        // }
    }
}
