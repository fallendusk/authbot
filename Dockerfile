FROM arm32v7/golang:1.8
ENV authbot_cmd iam
ENV authbot_prefix !
ENV authbot_role Members
ENV authbot_token

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["sh", "-c", "app -token ${authbot_token} -prefix ${authbot_prefix} -role ${authbot_role} -cmd ${authbot_cmd}"]
