FROM golang:alpine AS builder

WORKDIR /app
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOTOOLCHAIN=auto

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

RUN CGO_ENABLED=0 GOOS=linux go build -o /user_server ./cmd/user_server
RUN CGO_ENABLED=0 GOOS=linux go build -o /staff_server ./cmd/staff_server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /

COPY --from=builder /user_server /user_server
COPY --from=builder /staff_server /staff_server

EXPOSE 8080 8081

