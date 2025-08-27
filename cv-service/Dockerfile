FROM golang:1.25-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /cv-service
EXPOSE 8081
CMD [ "/cv-service" ]