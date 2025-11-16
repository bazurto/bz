def imageName = 'bzbuilder'
pipeline {
    agent any

    stages {
        stage('prepare') {
            steps {
                def dockerfile = '''
FROM golang:1.25
RUN apt-get update && apt-get upgrade
RUN apt-get install -y build-essential git && mkdir /work
WORKDIR /work
CMD ["bash"]
'''
                sh 'docker build -t '+ imageName + ' . -<<EOF\n' + dockerfile + '\nEOF'
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
