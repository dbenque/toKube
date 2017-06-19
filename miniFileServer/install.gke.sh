# FIRST you must update kubectl context with a command like:
#
#         gcloud container clusters get-credentials {clustername} --zone {zonename}
# 
# example: gcloud container clusters get-credentials cluster-1 --zone europe-west1-b
#
#
rm ./minifileserver >/dev/null 2>&1
GO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-s' .
if [ ! $? -eq 0 ]; then
 echo -e "go build failed"
 exit 1
fi
#docker build -t minifileserver:v0 .
docker build -t gcr.io/$DEVSHELL_PROJECT_ID/minifileserver:v0 .
if [ ! $? -eq 0 ]; then
 echo -e "docker build failed"
 exit 1
fi
echo "Pushing image to gcr.io/$DEVSHELL_PROJECT_ID"
gcloud docker -- push gcr.io/$DEVSHELL_PROJECT_ID/minifileserver:v0
if [ ! $? -eq 0 ]; then
 echo -e "docker push failed"
 exit 1
fi
echo "Deployement"
kubectl apply -f <(eval "echo \"$(< mfs.gke.yaml)\"")
