*Initalization*
**1. Run Dockerized Redis**
`docker run --name kitepay-redis -p 22000:6379 -d redis`
`--name kitepay-redis` sets the name of the docker container to kitepay-redis
`-p 22000:6379` runs the redis docker on localhost:22000
`-d` daemonize the container
Run a redis instance on port 22000 to use for the payment processor

**2. Run Docker Image**
`docker run --name kitepay-nano-pp -d --network host --env-file .env mitche50/kitepay:latest`
`-d` daemonize the container and run under kitepay-nano-pp name
`--network host` gives access to the localhost
`--env-file .env` uses the .env file located in the PWD to prepopulate environement variables
This must be run from the same directory as the .env file

*Update the Kitepay PP Docker Hub*
After testing a local copy of the file, create the docker image and push it to Docker Hub.

**1. Create Docker Image**
`docker build -t mitche50/kitepay .`
Creates a new docker image with the nano-pp:latest tag

**2. Push Docker Image to Docker Hub**
`docker push mitche50/kitepay:latest`
Sends the new docker image to docker hub