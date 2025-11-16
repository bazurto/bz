/**
SUDO_GID=0
JENKINS_HOME=/home/ubuntu
USER=ubuntu
CI=true
HOSTNAME=2d878b511e58
RUN_CHANGES_DISPLAY_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/15/display/redirect?page=changes
NODE_LABELS=built-in
HUDSON_URL=https://jenkins.bntso.com/
SHLVL=1
GIT_COMMIT=0bb209c9ffbf66b13c7b18b0d5958947acdd2027
HOME=/root
BUILD_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/15/
HUDSON_COOKIE=79b10856-d27f-4cb8-8ed4-8f86e2e3fe70
JENKINS_SERVER_COOKIE=durable-6a284b01db29572c51b85dbccc8b37a71d2b7d7b11a297ecd19be4dd9c87f4b1
WORKSPACE=/home/ubuntu/workspace/bazurto_bz_install-script
SUDO_UID=0
LOGNAME=ubuntu
NODE_NAME=built-in
RUN_ARTIFACTS_DISPLAY_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/15/display/redirect?page=artifacts
_=/usr/bin/sudo
STAGE_NAME=prepare
GIT_BRANCH=install-script
EXECUTOR_NUMBER=0
TERM=unknown
BUILD_DISPLAY_NAME=#15
RUN_TESTS_DISPLAY_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/15/display/redirect?page=tests
HUDSON_HOME=/home/ubuntu
JOB_BASE_NAME=install-script
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin
BUILD_ID=15
BUILD_TAG=jenkins-bazurto-bz-install-script-15
LANG=C.UTF-8
JENKINS_URL=https://jenkins.bntso.com/
JOB_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/
GIT_URL=https://github.com/bazurto/bz.git
SUDO_COMMAND=/usr/bin/java -jar /opt/jenkins.war
BUILD_NUMBER=15
JENKINS_NODE_COOKIE=3d580cf9-6496-4778-a460-2b6ea36b150e
SHELL=/bin/bash
RUN_DISPLAY_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/15/display/redirect
HUDSON_SERVER_COOKIE=77e96e85d98836ad
SUDO_USER=root
DOCKER_HOST=unix:///var/run/docker.sock
JOB_DISPLAY_URL=https://jenkins.bntso.com/job/bazurto/job/bz/job/install-script/display/redirect
JOB_NAME=bazurto/bz/install-script
PWD=/home/ubuntu/workspace/bazurto_bz_install-script
GIT_PREVIOUS_COMMIT=79b6fc6db817e9e293611843fe779cc46968309c
WORKSPACE_TMP=/home/ubuntu/workspace/bazurto_bz_install-script@tmp
BRANCH_NAME=install-script
*/
def imageName = 'bzbuilder'

def dockerfile = '''
FROM golang:1.25
RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y build-essential git && mkdir /work
WORKDIR /work
CMD ["bash"]
'''
def tmpDockerfile = null

pipeline {
    agent any

    environment {
        HOME = "${env.WORKSPACE}"
    }

    stages {
        stage('prepare') {
            steps {
                script {
                    tmpDockerfile = "${env.WORKSPACE}/Dockerfile.tmp1"
                    writeFile file: tmpDockerfile, text: dockerfile
                }
                sh "docker build -t ${imageName} . -f ${tmpDockerfile}"
            }
        }
        stage('Build') {
            steps {
                script {
                    def gid = sh(script: "id -g", returnStdout: true).trim()
                    def uid = sh(script: "id -u", returnStdout: true).trim()
                    sh "docker run --rm -u $uid:$gid -v \$PWD:/work -w /work $imageName make"
                }
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
