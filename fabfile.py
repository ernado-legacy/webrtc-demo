from fabric.api import run, local, env, cd


env.hosts = ['root@msk2.cydev.ru:122']
root = '/src/webrtc-demo'


def update():
    ver = local('git rev-parse HEAD', capture=True)
    with cd(root):
        remote_ver = run('git rev-parse HEAD')
        print('updating production to version %s' % ver)
        run('git reset --hard')
        run('git pull origin master')
        run('sed "s/VERSION/%s/g" Dockerfile.template > Dockerfile' % ver)
        run('docker build -t cydev/webrtc .')
        run('docker stop webrtc')
        run('docker rm webrtc')
        run('docker run -d --name webrtc cydev/webrtc')
