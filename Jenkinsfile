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
                    tmpDockerfile = new File("$WORKSPACE/Dockerfile.tmp")
                    tmpDockerfile.text = dockerfile
                }
                sh "docker build -t ${imageName} . -f ${tmpDockerfile.absolutePath}"
            }
        }
        stage('Build') {
            steps {
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
