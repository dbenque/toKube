rm ./minifileserver >/dev/null 2>&1
CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-s' .
if [ ! $? -eq 0 ]; then
 echo -e "go build failed"
 exit 1
fi

docker build -t minifileserver:v0 .
if [ ! $? -eq 0 ]; then
 echo -e "docker go build failed"
 exit 1
fi